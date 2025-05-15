package daemon

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/essentialkaos/ek/v13/errors"
	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/fmtutil"
	"github.com/essentialkaos/ek/v13/knf"
	"github.com/essentialkaos/ek/v13/log"
	"github.com/essentialkaos/ek/v13/mathutil"
	"github.com/essentialkaos/ek/v13/options"
	"github.com/essentialkaos/ek/v13/signal"
	"github.com/essentialkaos/ek/v13/support"
	"github.com/essentialkaos/ek/v13/support/deps"
	"github.com/essentialkaos/ek/v13/support/kernel"
	"github.com/essentialkaos/ek/v13/support/resources"
	"github.com/essentialkaos/ek/v13/system"
	"github.com/essentialkaos/ek/v13/system/sysctl"
	"github.com/essentialkaos/ek/v13/terminal"
	"github.com/essentialkaos/ek/v13/terminal/tty"
	"github.com/essentialkaos/ek/v13/timeutil"
	"github.com/essentialkaos/ek/v13/usage"

	knfv "github.com/essentialkaos/ek/v13/knf/validators"
	knff "github.com/essentialkaos/ek/v13/knf/validators/fs"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Basic service info
const (
	APP  = "swap-reaper"
	VER  = "0.0.2"
	DESC = "Service to periodically clean swap memory"
)

// Options
const (
	OPT_CONFIG   = "c:config"
	OPT_NO_COLOR = "nc:no-color"
	OPT_HELP     = "h:help"
	OPT_VER      = "v:version"

	OPT_VERB_VER = "vv:verbose-version"
)

// Configuration file properties
const (
	LIMITS_MAX_LA   = "limits:max-la"
	LIMITS_MAX_WAIT = "limits:max-wait"
	LOG_DIR         = "log:dir"
	LOG_FILE        = "log:file"
	LOG_PERMS       = "log:perms"
	LOG_LEVEL       = "log:level"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// optMap contains information about all supported options
var optMap = options.Map{
	OPT_CONFIG:   {Value: "/etc/swap-reaper.knf"},
	OPT_NO_COLOR: {Type: options.BOOL},
	OPT_HELP:     {Type: options.BOOL},
	OPT_VER:      {Type: options.MIXED},

	OPT_VERB_VER: {Type: options.BOOL},
}

// color tags for app name and version
var colorTagApp, colorTagVer string

// ////////////////////////////////////////////////////////////////////////////////// //

// Run is main daemon function
func Run(gitRev string, gomod []byte) {
	preConfigureUI()

	_, errs := options.Parse(optMap)

	if !errs.IsEmpty() {
		terminal.Error("Options parsing errors:")
		terminal.Error(errs.Error("- "))
		os.Exit(1)
	}

	configureUI()

	switch {
	case options.GetB(OPT_VER):
		genAbout(gitRev).Print(options.GetS(OPT_VER))
		os.Exit(0)
	case options.GetB(OPT_HELP):
		genUsage().Print()
		os.Exit(0)
	case options.GetB(OPT_VERB_VER):
		support.Collect(APP, VER).
			WithRevision(gitRev).
			WithDeps(deps.Extract(gomod)).
			WithResources(resources.Collect()).
			WithKernel(kernel.Collect("vm.swappiness")).
			Print()
		os.Exit(0)
	}

	err := errors.Chain(
		checkUser,
		loadConfig,
		validateConfig,
		registerSignalHandlers,
		setupLogger,
	)

	if err != nil {
		terminal.Error(err)
		os.Exit(1)
	}

	log.Divider()
	log.Aux("%s %s starting…", APP, VER)

	err = start()

	if err != nil {
		log.Crit(err.Error())
		os.Exit(1)
	}
}

// ////////////////////////////////////////////////////////////////////////////////// //

// preConfigureUI preconfigures UI based on information about user terminal
func preConfigureUI() {
	if !tty.IsTTY() || tty.IsSystemd() {
		fmtc.DisableColors = true
	}

	switch {
	case fmtc.IsTrueColorSupported():
		colorTagApp, colorTagVer = "{*}{#00AFFF}", "{#00AFFF}"
	case fmtc.Is256ColorsSupported():
		colorTagApp, colorTagVer = "{*}{#39}", "{#39}"
	default:
		colorTagApp, colorTagVer = "{*}{c}", "{c}"
	}
}

// configureUI configures user interface
func configureUI() {
	if options.GetB(OPT_NO_COLOR) {
		fmtc.DisableColors = true
	}
}

// checkUser checks if current user is root
func checkUser() error {
	user, err := system.CurrentUser()

	if err != nil {
		return fmt.Errorf("Can't get info about current user: %v", err)
	}

	if !user.IsRoot() {
		return fmt.Errorf("You must run this daemon as super user (root)")
	}

	return nil
}

// loadConfig loads configuration file
func loadConfig() error {
	err := knf.Global(options.GetS(OPT_CONFIG))

	if err != nil {
		return fmt.Errorf("Can't load configuration: %w", err)
	}

	return nil
}

// validateConfig validates configuration file values
func validateConfig() error {
	errs := knf.Validate([]*knf.Validator{
		{LIMITS_MAX_LA, knfv.TypeFloat, nil},
		{LIMITS_MAX_LA, knfv.Greater, 0.1},
		{LIMITS_MAX_WAIT, knfv.Greater, 1},
		{LIMITS_MAX_WAIT, knfv.Less, 24 * 3600},
		{LOG_DIR, knff.Perms, "DWX"},
		{LOG_LEVEL, knfv.SetToAnyIgnoreCase, []string{
			"debug", "info", "warn", "error", "crit",
		}},
	})

	if len(errs) != 0 {
		return fmt.Errorf("Configuration file validation error: %w", errs[0])
	}

	return nil
}

// registerSignalHandlers registers signal handlers
func registerSignalHandlers() error {
	signal.Handlers{
		signal.TERM: termSignalHandler,
		signal.INT:  intSignalHandler,
		signal.HUP:  hupSignalHandler,
	}.TrackAsync()

	return nil
}

// setupLogger configures logger subsystem
func setupLogger() error {
	err := log.Set(knf.GetS(LOG_FILE), knf.GetM(LOG_PERMS, 0644))

	if err != nil {
		return err
	}

	err = log.MinLevel(knf.GetS(LOG_LEVEL))

	if err != nil {
		return err
	}

	return nil
}

// start starts daemon
func start() error {
	mem, err := system.GetMemUsage()

	if err != nil {
		return fmt.Errorf("Can't get memory usage info: %v", err)
	}

	if mem.SwapTotal == 0 {
		return fmt.Errorf("Swap is disabled, nothing to do…")
	}

	swappiness, err := sysctl.GetI("vm.swappiness")

	if err != nil {
		return fmt.Errorf("Can't read swappiness configuration: %v", err)
	}

	if swappiness > 30 {
		log.Warn("The kernel parameter 'vm.swappiness' is too high! A value between 30 and 5 is recommended.")
	}

	log.Aux("Initialization finished, monitoring system swap…")

	checkLoop(swappiness)

	return nil
}

// checkLoop is function with check loop
func checkLoop(swappiness int) {
	var err error
	var mem *system.MemUsage
	var maxMem float64
	var lastCheck time.Time

	maxWait := time.Duration(knf.GetI(LIMITS_MAX_WAIT, 10)) * time.Minute
	maxLA := knf.GetF(LIMITS_MAX_LA, 0.1)

	for range time.NewTicker(time.Minute).C {
		mem, err = system.GetMemUsage()

		if err != nil {
			log.Error("Can't get system memory usage: %v", err)
			continue
		}

		if maxMem == 0 {
			maxMem = float64(mem.MemTotal) - mathutil.FromPerc(float64(swappiness), float64(mem.MemTotal))
		}

		log.Debug(
			"Memory: %s / %s | Swap: %s / %s | Swappiness: %s (≥ %s)",
			fmtutil.PrettySize(mem.MemUsed), fmtutil.PrettySize(mem.MemTotal),
			fmtutil.PrettySize(mem.SwapUsed), fmtutil.PrettySize(mem.SwapTotal),
			fmtutil.PrettyPerc(float64(swappiness)), fmtutil.PrettySize(maxMem),
		)

		if mem.SwapUsed == 0 {
			continue
		}

		if mem.SwapUsed+mem.MemUsed > uint64(maxMem) {
			log.Warn(
				"Not enough memory to clean up swap: swap (%s) + used (%s) > %s (swappiness %s)",
				fmtutil.PrettySize(mem.MemUsed), fmtutil.PrettySize(mem.SwapUsed),
				fmtutil.PrettySize(maxMem), fmtutil.PrettyPerc(float64(swappiness)),
			)
			continue
		}

		la, err := system.GetLA()

		if err != nil {
			log.Error("Can't check system LA: %v", err)
			continue
		}

		if la.Min1 >= maxLA && maxWait != time.Minute {
			if lastCheck.IsZero() {
				lastCheck = time.Now()
				log.Warn(
					"System LA is too big (%s ≥ %s), cleaning is delayed (max wait: %d min)",
					fmtutil.PrettyNum(la.Min1), fmtutil.PrettyNum(maxLA),
					knf.GetI(LIMITS_MAX_WAIT, 10),
				)
				continue
			} else {
				if time.Since(lastCheck) < maxWait {
					continue
				}

				log.Warn(
					"System LA is too big (%s ≥ %s), but we've reached the maximum wait limit (%d min). Clean anyway…",
					fmtutil.PrettyNum(la.Min1), fmtutil.PrettyNum(maxLA),
					knf.GetI(LIMITS_MAX_WAIT, 10),
				)
			}
		}

		log.Info("Found swap to clean (%s), cleaning…", fmtutil.PrettySize(mem.SwapUsed))

		start := time.Now()
		err = cleanSwap()

		if err != nil {
			log.Error(err.Error())
			continue
		}

		newMem, err := system.GetMemUsage()

		if err != nil {
			log.Info(
				"Data successfully moved from swap to memory (took %s)",
				timeutil.ShortDuration(time.Since(start), true),
			)
		} else {
			log.Info(
				"Data successfully moved from swap to memory (took %s). Memory: %s / %s (%s)",
				timeutil.ShortDuration(time.Since(start), true),
				fmtutil.PrettySize(newMem.MemUsed), fmtutil.PrettySize(newMem.MemTotal),
				fmtutil.PrettyPerc(mathutil.Perc(newMem.MemUsed, newMem.MemTotal)),
			)
		}

		lastCheck = time.Time{}
	}
}

// cleanSwap moves data from swap to memory using swapoff & swapon
func cleanSwap() error {
	cmdOff := exec.Command("swapoff", "-a")
	err := cmdOff.Run()

	if err != nil {
		ec := cmdOff.ProcessState.ExitCode()
		return fmt.Errorf("Can't disable swap using swapoff: swapoff exited with code %d", ec)
	}

	cmdOn := exec.Command("swapon", "-a")
	err = cmdOn.Run()

	if err != nil {
		ec := cmdOff.ProcessState.ExitCode()
		return fmt.Errorf("Can't enable swap back using swapon: swapon exited with code %d", ec)
	}

	return nil
}

// intSignalHandler is INT signal handler
func intSignalHandler() {
	log.Aux("Received INT signal, shutdown…")
	log.Flush()
	os.Exit(0)
}

// termSignalHandler is TERM signal handler
func termSignalHandler() {
	log.Aux("Received TERM signal, shutdown…")
	log.Flush()
	os.Exit(0)
}

// hupSignalHandler is HUP signal handler
func hupSignalHandler() {
	log.Info("Received HUP signal, log will be reopened…")
	log.Reopen()
	log.Info("Log reopened by HUP signal")
}

// ////////////////////////////////////////////////////////////////////////////////// //

// genUsage generates usage info
func genUsage() *usage.Info {
	info := usage.NewInfo()

	info.AddOption(OPT_CONFIG, "Path to configuration file", "file")
	info.AddOption(OPT_NO_COLOR, "Disable colors in output")
	info.AddOption(OPT_HELP, "Show this help message")
	info.AddOption(OPT_VER, "Show version")

	return info
}

// genAbout generates info about version
func genAbout(gitRev string) *usage.About {
	about := &usage.About{
		App:     APP,
		Version: VER,
		Desc:    DESC,
		Year:    2009,
		Owner:   "ESSENTIAL KAOS",

		AppNameColorTag: colorTagApp,
		VersionColorTag: colorTagVer,
		DescSeparator:   "{s}—{!}",

		BugTracker: "https://github.com/essentialkaos/swap-reaper/issues",
		License:    "Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>",
	}

	if gitRev != "" {
		about.Build = "git:" + gitRev
	}

	return about
}
