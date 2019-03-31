package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/Jguer/yay/v9/conf"
	"github.com/Jguer/yay/v9/dep"
	"github.com/Jguer/yay/v9/generic"
	"github.com/Jguer/yay/v9/generic/exe"
	"github.com/Jguer/yay/v9/install/keys"
	gosrc "github.com/Morganamilo/go-srcinfo"
	alpm "github.com/jguer/go-alpm"
)

const gitEmptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// Install handles package installs
func install(parser *arguments) error {
	var err error
	var incompatible generic.StringSet
	var do *depOrder

	var aurUp upSlice
	var repoUp upSlice

	var srcinfos map[string]*gosrc.Srcinfo

	warnings := &aurWarnings{}
	removeMake := false

	if mode == modeAny || mode == modeRepo {
		if config.CombinedUpgrade {
			if parser.existsArg("y", "refresh") {
				err = earlyRefresh(parser)
				if err != nil {
					return fmt.Errorf("Error refreshing databases")
				}
			}
		} else if parser.existsArg("y", "refresh") || parser.existsArg("u", "sysupgrade") || len(parser.targets) > 0 {
			err = earlyPacmanCall(parser)
			if err != nil {
				return err
			}
		}
	}

	//we may have done -Sy, our handle now has an old
	//database.
	err = initAlpmHandle()
	if err != nil {
		return err
	}

	_, _, localNames, remoteNames, err := filterPackages()
	if err != nil {
		return err
	}

	remoteNamesCache := generic.SliceToStringSet(remoteNames)
	localNamesCache := generic.SliceToStringSet(localNames)

	requestTargets := parser.copy().targets

	//create the arguments to pass for the repo install
	arguments := parser.copy()
	arguments.delArg("asdeps", "asdep")
	arguments.delArg("asexplicit", "asexp")
	arguments.op = "S"
	arguments.clearTargets()

	if mode == modeAUR {
		arguments.delArg("u", "sysupgrade")
	}

	//if we are doing -u also request all packages needing update
	if parser.existsArg("u", "sysupgrade") {
		aurUp, repoUp, err = upList(warnings)
		if err != nil {
			return err
		}

		warnings.print()

		ignore, aurUp, err := upgradePkgs(aurUp, repoUp)
		if err != nil {
			return err
		}

		for _, up := range repoUp {
			if !ignore.Get(up.Name) {
				requestTargets = append(requestTargets, up.Name)
				parser.addTarget(up.Name)
			}
		}

		for up := range aurUp {
			requestTargets = append(requestTargets, "aur/"+up)
			parser.addTarget("aur/" + up)
		}

		value, _, exists := cmdArgs.getArg("ignore")

		if len(ignore) > 0 {
			ignoreStr := strings.Join(ignore.ToSlice(), ",")
			if exists {
				ignoreStr += "," + value
			}
			arguments.options["ignore"] = ignoreStr
		}
	}

	targets := generic.SliceToStringSet(parser.targets)

	dp, err := getDepPool(requestTargets, warnings)
	if err != nil {
		return err
	}

	err = dp.CheckMissing()
	if err != nil {
		return err
	}

	if len(dp.Aur) == 0 {
		if !config.CombinedUpgrade {
			if parser.existsArg("u", "sysupgrade") {
				fmt.Println(" there is nothing to do")
			}
			return nil
		}

		parser.op = "S"
		parser.delArg("y", "refresh")
		parser.options["ignore"] = arguments.options["ignore"]
		return exe.Show(passToPacman(parser))
	}

	if len(dp.Aur) > 0 && os.Geteuid() == 0 {
		return fmt.Errorf(generic.Bold(generic.Red(generic.Arrow)) + " Refusing to install AUR Packages as root, Aborting.")
	}

	conflicts, err := dp.CheckConflicts()
	if err != nil {
		return err
	}

	do = getDepOrder(dp)
	if err != nil {
		return err
	}

	for _, pkg := range do.Repo {
		arguments.addTarget(pkg.DB().Name() + "/" + pkg.Name())
	}

	for _, pkg := range dp.Groups {
		arguments.addTarget(pkg)
	}

	if len(do.Aur) == 0 && len(arguments.targets) == 0 && (!parser.existsArg("u", "sysupgrade") || mode == modeAUR) {
		fmt.Println(" there is nothing to do")
		return nil
	}

	do.Print()
	fmt.Println()

	if do.HasMake() {
		switch config.RemoveMake {
		case "yes":
			removeMake = true
		case "no":
			removeMake = false
		default:
			removeMake = generic.ContinueTask("Remove make dependencies after install?", false)
		}
	}

	if config.CleanMenu {
		if anyExistInCache(do.Aur) {
			askClean := pkgbuildNumberMenu(do.Aur, remoteNamesCache)
			toClean, err := cleanNumberMenu(do.Aur, remoteNamesCache, askClean)
			if err != nil {
				return err
			}

			cleanBuilds(toClean)
		}
	}

	toSkip := pkgbuildsToSkip(do.Aur, targets)
	cloned, err := downloadPkgbuilds(do.Aur, toSkip, config.BuildDir)
	if err != nil {
		return err
	}

	var toDiff []dep.Base
	var toEdit []dep.Base

	if config.DiffMenu {
		pkgbuildNumberMenu(do.Aur, remoteNamesCache)
		toDiff, err = diffNumberMenu(do.Aur, remoteNamesCache)
		if err != nil {
			return err
		}

		if len(toDiff) > 0 {
			err = showPkgbuildDiffs(toDiff, cloned)
			if err != nil {
				return err
			}
		}
	}

	if len(toDiff) > 0 {
		oldValue := config.NoConfirm
		config.NoConfirm = false
		fmt.Println()
		if !generic.ContinueTask(generic.Bold(generic.Green("Proceed with install?")), true) {
			return fmt.Errorf("Aborting due to user")
		}
		config.NoConfirm = oldValue
	}

	err = mergePkgbuilds(do.Aur)
	if err != nil {
		return err
	}

	srcinfos, err = parseSrcinfoFiles(do.Aur, true)
	if err != nil {
		return err
	}

	if config.EditMenu {
		pkgbuildNumberMenu(do.Aur, remoteNamesCache)
		toEdit, err = editNumberMenu(do.Aur, remoteNamesCache)
		if err != nil {
			return err
		}

		if len(toEdit) > 0 {
			err = editPkgbuilds(toEdit, srcinfos)
			if err != nil {
				return err
			}
		}
	}

	if len(toEdit) > 0 {
		oldValue := config.NoConfirm
		config.NoConfirm = false
		fmt.Println()
		if !generic.ContinueTask(generic.Bold(generic.Green("Proceed with install?")), true) {
			return fmt.Errorf("Aborting due to user")
		}
		config.NoConfirm = oldValue
	}

	incompatible, err = getIncompatible(do.Aur, srcinfos)
	if err != nil {
		return err
	}

	if config.PGPFetch {
		err = keys.CheckPgpKeys(do.Aur, srcinfos)
		if err != nil {
			return err
		}
	}

	if !config.CombinedUpgrade {
		arguments.delArg("u", "sysupgrade")
	}

	if len(arguments.targets) > 0 || arguments.existsArg("u") {
		err := exe.Show(passToPacman(arguments))
		if err != nil {
			return fmt.Errorf("Error installing repo packages")
		}

		depArguments := makeArguments()
		depArguments.addArg("D", "asdeps")
		expArguments := makeArguments()
		expArguments.addArg("D", "asexplicit")

		for _, pkg := range do.Repo {
			if !dp.Explicit.Get(pkg.Name()) && !localNamesCache.Get(pkg.Name()) && !remoteNamesCache.Get(pkg.Name()) {
				depArguments.addTarget(pkg.Name())
				continue
			}

			if parser.existsArg("asdeps", "asdep") && dp.Explicit.Get(pkg.Name()) {
				depArguments.addTarget(pkg.Name())
			} else if parser.existsArg("asexp", "asexplicit") && dp.Explicit.Get(pkg.Name()) {
				expArguments.addTarget(pkg.Name())
			}
		}

		if len(depArguments.targets) > 0 {
			_, stderr, err := exe.Capture(passToPacman(depArguments))
			if err != nil {
				return fmt.Errorf("%s%s", stderr, err)
			}
		}

		if len(expArguments.targets) > 0 {
			_, stderr, err := exe.Capture(passToPacman(expArguments))
			if err != nil {
				return fmt.Errorf("%s%s", stderr, err)
			}
		}
	}

	go updateCompletion(false)

	err = downloadPkgbuildsSources(do.Aur, incompatible)
	if err != nil {
		return err
	}

	err = buildInstallPkgbuilds(dp, do, srcinfos, parser, incompatible, conflicts)
	if err != nil {
		return err
	}

	if removeMake {
		removeArguments := makeArguments()
		removeArguments.addArg("R", "u")

		for _, pkg := range do.getMake() {
			removeArguments.addTarget(pkg)
		}

		oldValue := config.NoConfirm
		config.NoConfirm = true
		err = exe.Show(passToPacman(removeArguments))
		config.NoConfirm = oldValue

		if err != nil {
			return err
		}
	}

	if config.CleanAfter {
		cleanAfter(do.Aur)
	}

	return nil
}

func inRepos(syncDB alpm.DBList, pkg string) bool {
	target := toTarget(pkg)

	if target.DB == "aur" {
		return false
	} else if target.DB != "" {
		return true
	}

	previousHideMenus := hideMenus
	hideMenus = false
	_, err := syncDB.FindSatisfier(target.DepString())
	hideMenus = previousHideMenus
	if err == nil {
		return true
	}

	return !syncDB.FindGroupPkgs(target.Name).Empty()
}

func earlyPacmanCall(parser *arguments) error {
	arguments := parser.copy()
	arguments.op = "S"
	targets := parser.targets
	parser.clearTargets()
	arguments.clearTargets()

	syncDB, err := alpmHandle.SyncDBs()
	if err != nil {
		return err
	}

	if mode == modeRepo {
		arguments.targets = targets
	} else {
		//separate aur and repo targets
		for _, target := range targets {
			if inRepos(syncDB, target) {
				arguments.addTarget(target)
			} else {
				parser.addTarget(target)
			}
		}
	}

	if parser.existsArg("y", "refresh") || parser.existsArg("u", "sysupgrade") || len(arguments.targets) > 0 {
		err = exe.Show(passToPacman(arguments))
		if err != nil {
			return fmt.Errorf("Error installing repo packages")
		}
	}

	return nil
}

func earlyRefresh(parser *arguments) error {
	arguments := parser.copy()
	parser.delArg("y", "refresh")
	arguments.delArg("u", "sysupgrade")
	arguments.delArg("s", "search")
	arguments.delArg("i", "info")
	arguments.delArg("l", "list")
	arguments.clearTargets()
	return exe.Show(passToPacman(arguments))
}

func getIncompatible(bases []dep.Base, srcinfos map[string]*gosrc.Srcinfo) (generic.StringSet, error) {
	incompatible := make(generic.StringSet)
	basesMap := make(map[string]dep.Base)
	alpmArch, err := alpmHandle.Arch()
	if err != nil {
		return nil, err
	}

nextpkg:
	for _, base := range bases {
		for _, arch := range srcinfos[base.Pkgbase()].Arch {
			if arch == "any" || arch == alpmArch {
				continue nextpkg
			}
		}

		incompatible.Set(base.Pkgbase())
		basesMap[base.Pkgbase()] = base
	}

	if len(incompatible) > 0 {
		fmt.Println()
		fmt.Print(generic.Bold(generic.Yellow(generic.Arrow)) + " The following packages are not compatible with your architecture:")
		for pkg := range incompatible {
			fmt.Print("  " + generic.Cyan(basesMap[pkg].String()))
		}

		fmt.Println()

		if !generic.ContinueTask("Try to build them anyway?", true) {
			return nil, fmt.Errorf("Aborting due to user")
		}
	}

	return incompatible, nil
}

func parsePackageList(dir string) (map[string]string, string, error) {
	stdout, stderr, err := exe.Capture(passToMakepkg(dir, "--packagelist"))

	if err != nil {
		return nil, "", fmt.Errorf("%s%s", stderr, err)
	}

	var version string
	lines := strings.Split(stdout, "\n")
	pkgdests := make(map[string]string)

	for _, line := range lines {
		if line == "" {
			continue
		}

		fileName := filepath.Base(line)
		split := strings.Split(fileName, "-")

		if len(split) < 4 {
			return nil, "", fmt.Errorf("Can not find package name : %s", split)
		}

		// pkgname-pkgver-pkgrel-arch.pkgext
		// This assumes 3 dashes after the pkgname, Will cause an error
		// if the PKGEXT contains a dash. Please no one do that.
		pkgname := strings.Join(split[:len(split)-3], "-")
		version = strings.Join(split[len(split)-3:len(split)-1], "-")
		pkgdests[pkgname] = line
	}

	return pkgdests, version, nil
}

func anyExistInCache(bases []dep.Base) bool {
	for _, base := range bases {
		pkg := base.Pkgbase()
		dir := filepath.Join(config.BuildDir, pkg)

		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			return true
		}
	}

	return false
}

func pkgbuildNumberMenu(bases []dep.Base, installed generic.StringSet) bool {
	toPrint := ""
	askClean := false

	for n, base := range bases {
		pkg := base.Pkgbase()
		dir := filepath.Join(config.BuildDir, pkg)

		toPrint += fmt.Sprintf(generic.Magenta("%3d")+" %-40s", len(bases)-n,
			generic.Bold(base.String()))

		anyInstalled := false
		for _, b := range base {
			anyInstalled = anyInstalled || installed.Get(b.Name)
		}

		if anyInstalled {
			toPrint += generic.Bold(generic.Green(" (Installed)"))
		}

		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			toPrint += generic.Bold(generic.Green(" (Build Files Exist)"))
			askClean = true
		}

		toPrint += "\n"
	}

	fmt.Print(toPrint)

	return askClean
}

func cleanNumberMenu(bases []dep.Base, installed generic.StringSet, hasClean bool) ([]dep.Base, error) {
	toClean := make([]dep.Base, 0)

	if !hasClean {
		return toClean, nil
	}

	fmt.Println(generic.Bold(generic.Green(generic.Arrow + " Packages to cleanBuild?")))
	fmt.Println(generic.Bold(generic.Green(generic.Arrow) + generic.Cyan(" [N]one ") + "[A]ll [Ab]ort [I]nstalled [No]tInstalled or (1 2 3, 1-3, ^4)"))
	fmt.Print(generic.Bold(generic.Green(generic.Arrow + " ")))
	cleanInput, err := getInput(config.AnswerClean)
	if err != nil {
		return nil, err
	}

	cInclude, cExclude, cOtherInclude, cOtherExclude := parseNumberMenu(cleanInput)
	cIsInclude := len(cExclude) == 0 && len(cOtherExclude) == 0

	if cOtherInclude.Get("abort") || cOtherInclude.Get("ab") {
		return nil, fmt.Errorf("Aborting due to user")
	}

	if !cOtherInclude.Get("n") && !cOtherInclude.Get("none") {
		for i, base := range bases {
			pkg := base.Pkgbase()
			anyInstalled := false
			for _, b := range base {
				anyInstalled = anyInstalled || installed.Get(b.Name)
			}

			dir := filepath.Join(config.BuildDir, pkg)
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				continue
			}

			if !cIsInclude && cExclude.Get(len(bases)-i) {
				continue
			}

			if anyInstalled && (cOtherInclude.Get("i") || cOtherInclude.Get("installed")) {
				toClean = append(toClean, base)
				continue
			}

			if !anyInstalled && (cOtherInclude.Get("no") || cOtherInclude.Get("notinstalled")) {
				toClean = append(toClean, base)
				continue
			}

			if cOtherInclude.Get("a") || cOtherInclude.Get("all") {
				toClean = append(toClean, base)
				continue
			}

			if cIsInclude && (cInclude.Get(len(bases)-i) || cOtherInclude.Get(pkg)) {
				toClean = append(toClean, base)
				continue
			}

			if !cIsInclude && (!cExclude.Get(len(bases)-i) && !cOtherExclude.Get(pkg)) {
				toClean = append(toClean, base)
				continue
			}
		}
	}

	return toClean, nil
}

func editNumberMenu(bases []dep.Base, installed generic.StringSet) ([]dep.Base, error) {
	return editDiffNumberMenu(bases, installed, false)
}

func diffNumberMenu(bases []dep.Base, installed generic.StringSet) ([]dep.Base, error) {
	return editDiffNumberMenu(bases, installed, true)
}

func editDiffNumberMenu(bases []dep.Base, installed generic.StringSet, diff bool) ([]dep.Base, error) {
	toEdit := make([]dep.Base, 0)
	var editInput string
	var err error

	if diff {
		fmt.Println(generic.Bold(generic.Green(generic.Arrow + " Diffs to show?")))
		fmt.Println(generic.Bold(generic.Green(generic.Arrow) + generic.Cyan(" [N]one ") + "[A]ll [Ab]ort [I]nstalled [No]tInstalled or (1 2 3, 1-3, ^4)"))
		fmt.Print(generic.Bold(generic.Green(generic.Arrow + " ")))
		editInput, err = getInput(config.AnswerDiff)
		if err != nil {
			return nil, err
		}
	} else {
		fmt.Println(generic.Bold(generic.Green(generic.Arrow + " PKGBUILDs to edit?")))
		fmt.Println(generic.Bold(generic.Green(generic.Arrow) + generic.Cyan(" [N]one ") + "[A]ll [Ab]ort [I]nstalled [No]tInstalled or (1 2 3, 1-3, ^4)"))
		fmt.Print(generic.Bold(generic.Green(generic.Arrow + " ")))
		editInput, err = getInput(config.AnswerEdit)
		if err != nil {
			return nil, err
		}
	}

	eInclude, eExclude, eOtherInclude, eOtherExclude := parseNumberMenu(editInput)
	eIsInclude := len(eExclude) == 0 && len(eOtherExclude) == 0

	if eOtherInclude.Get("abort") || eOtherInclude.Get("ab") {
		return nil, fmt.Errorf("Aborting due to user")
	}

	if !eOtherInclude.Get("n") && !eOtherInclude.Get("none") {
		for i, base := range bases {
			pkg := base.Pkgbase()
			anyInstalled := false
			for _, b := range base {
				anyInstalled = anyInstalled || installed.Get(b.Name)
			}

			if !eIsInclude && eExclude.Get(len(bases)-i) {
				continue
			}

			if anyInstalled && (eOtherInclude.Get("i") || eOtherInclude.Get("installed")) {
				toEdit = append(toEdit, base)
				continue
			}

			if !anyInstalled && (eOtherInclude.Get("no") || eOtherInclude.Get("notinstalled")) {
				toEdit = append(toEdit, base)
				continue
			}

			if eOtherInclude.Get("a") || eOtherInclude.Get("all") {
				toEdit = append(toEdit, base)
				continue
			}

			if eIsInclude && (eInclude.Get(len(bases)-i) || eOtherInclude.Get(pkg)) {
				toEdit = append(toEdit, base)
			}

			if !eIsInclude && (!eExclude.Get(len(bases)-i) && !eOtherExclude.Get(pkg)) {
				toEdit = append(toEdit, base)
			}
		}
	}

	return toEdit, nil
}

func showPkgbuildDiffs(bases []dep.Base, cloned generic.StringSet) error {
	for _, base := range bases {
		pkg := base.Pkgbase()
		dir := filepath.Join(config.BuildDir, pkg)
		if shouldUseGit(dir) {
			start := "HEAD"

			if cloned.Get(pkg) {
				start = gitEmptyTree
			} else {
				hasDiff, err := gitHasDiff(config.BuildDir, pkg)
				if err != nil {
					return err
				}

				if !hasDiff {
					fmt.Printf("%s %s: %s\n", generic.Bold(generic.Yellow(generic.Arrow)), generic.Cyan(base.String()), generic.Bold("No changes -- skipping"))
					continue
				}
			}

			args := []string{"diff", start + "..HEAD@{upstream}", "--src-prefix", dir + "/", "--dst-prefix", dir + "/", "--", ".", ":(exclude).SRCINFO"}
			if conf.UseColor {
				args = append(args, "--color=always")
			} else {
				args = append(args, "--color=never")
			}
			err := exe.Show(passToGit(dir, args...))
			if err != nil {
				return err
			}
		} else {
			args := []string{"diff"}
			if conf.UseColor {
				args = append(args, "--color=always")
			} else {
				args = append(args, "--color=never")
			}
			args = append(args, "--no-index", "/var/empty", dir)
			// git always returns 1. why? I have no idea
			exe.Show(passToGit(dir, args...))
		}
	}

	return nil
}

func editPkgbuilds(bases []dep.Base, srcinfos map[string]*gosrc.Srcinfo) error {
	pkgbuilds := make([]string, 0, len(bases))
	for _, base := range bases {
		pkg := base.Pkgbase()
		dir := filepath.Join(config.BuildDir, pkg)
		pkgbuilds = append(pkgbuilds, filepath.Join(dir, "PKGBUILD"))

		for _, splitPkg := range srcinfos[pkg].SplitPackages() {
			if splitPkg.Install != "" {
				pkgbuilds = append(pkgbuilds, filepath.Join(dir, splitPkg.Install))
			}
		}
	}

	if len(pkgbuilds) > 0 {
		editor, editorArgs := editor()
		editorArgs = append(editorArgs, pkgbuilds...)
		editcmd := exec.Command(editor, editorArgs...)
		editcmd.Stdin, editcmd.Stdout, editcmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		err := editcmd.Run()
		if err != nil {
			return fmt.Errorf("Editor did not exit successfully, Aborting: %s", err)
		}
	}

	return nil
}

func parseSrcinfoFiles(bases []dep.Base, errIsFatal bool) (map[string]*gosrc.Srcinfo, error) {
	srcinfos := make(map[string]*gosrc.Srcinfo)
	for k, base := range bases {
		pkg := base.Pkgbase()
		dir := filepath.Join(config.BuildDir, pkg)

		str := generic.Bold(generic.Cyan("::") + " Parsing SRCINFO (%d/%d): %s\n")
		fmt.Printf(str, k+1, len(bases), generic.Cyan(base.String()))

		pkgbuild, err := gosrc.ParseFile(filepath.Join(dir, ".SRCINFO"))
		if err != nil {
			if !errIsFatal {
				fmt.Fprintf(os.Stderr, "failed to parse %s -- skipping: %s\n", base.String(), err)
				continue
			}
			return nil, fmt.Errorf("failed to parse %s: %s", base.String(), err)
		}

		srcinfos[pkg] = pkgbuild
	}

	return srcinfos, nil
}

func pkgbuildsToSkip(bases []dep.Base, targets generic.StringSet) generic.StringSet {
	toSkip := make(generic.StringSet)

	for _, base := range bases {
		isTarget := false
		for _, pkg := range base {
			isTarget = isTarget || targets.Get(pkg.Name)
		}

		if (config.ReDownload == "yes" && isTarget) || config.ReDownload == "all" {
			continue
		}

		dir := filepath.Join(config.BuildDir, base.Pkgbase(), ".SRCINFO")
		pkgbuild, err := gosrc.ParseFile(dir)

		if err == nil {
			if alpm.VerCmp(pkgbuild.Version(), base.Version()) >= 0 {
				toSkip.Set(base.Pkgbase())
			}
		}
	}

	return toSkip
}

func mergePkgbuilds(bases []dep.Base) error {
	for _, base := range bases {
		if shouldUseGit(filepath.Join(config.BuildDir, base.Pkgbase())) {
			err := gitMerge(config.BuildDir, base.Pkgbase())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func downloadPkgbuilds(bases []dep.Base, toSkip generic.StringSet, buildDir string) (generic.StringSet, error) {
	cloned := make(generic.StringSet)
	downloaded := 0
	var wg sync.WaitGroup
	var mux sync.Mutex
	var errs generic.MultiError

	download := func(k int, base dep.Base) {
		defer wg.Done()
		pkg := base.Pkgbase()

		if toSkip.Get(pkg) {
			mux.Lock()
			downloaded++
			str := generic.Bold(generic.Cyan("::") + " PKGBUILD up to date, Skipping (%d/%d): %s\n")
			fmt.Printf(str, downloaded, len(bases), generic.Cyan(base.String()))
			mux.Unlock()
			return
		}

		if shouldUseGit(filepath.Join(config.BuildDir, pkg)) {
			clone, err := gitDownload(config.AURURL+"/"+pkg+".git", buildDir, pkg)
			if err != nil {
				errs.Add(err)
				return
			}
			if clone {
				mux.Lock()
				cloned.Set(pkg)
				mux.Unlock()
			}
		} else {
			err := downloadAndUnpack(config.AURURL+base.URLPath(), buildDir)
			if err != nil {
				errs.Add(err)
				return
			}
		}

		mux.Lock()
		downloaded++
		str := generic.Bold(generic.Cyan("::") + " Downloaded PKGBUILD (%d/%d): %s\n")
		fmt.Printf(str, downloaded, len(bases), generic.Cyan(base.String()))
		mux.Unlock()
	}

	count := 0
	for k, base := range bases {
		wg.Add(1)
		go download(k, base)
		count++
		if count%25 == 0 {
			wg.Wait()
		}
	}

	wg.Wait()

	return cloned, errs.Return()
}

func downloadPkgbuildsSources(bases []dep.Base, incompatible generic.StringSet) (err error) {
	for _, base := range bases {
		pkg := base.Pkgbase()
		dir := filepath.Join(config.BuildDir, pkg)
		args := []string{"--verifysource", "-Ccf"}

		if incompatible.Get(pkg) {
			args = append(args, "--ignorearch")
		}

		err = exe.Show(passToMakepkg(dir, args...))
		if err != nil {
			return fmt.Errorf("Error downloading sources: %s", generic.Cyan(base.String()))
		}
	}

	return
}

func buildInstallPkgbuilds(dp *depPool, do *depOrder, srcinfos map[string]*gosrc.Srcinfo, parser *arguments, incompatible generic.StringSet, conflicts generic.MapStringSet) error {
	for _, base := range do.Aur {
		pkg := base.Pkgbase()
		dir := filepath.Join(config.BuildDir, pkg)
		built := true

		srcinfo := srcinfos[pkg]

		args := []string{"--nobuild", "-fC"}

		if incompatible.Get(pkg) {
			args = append(args, "--ignorearch")
		}

		//pkgver bump
		err := exe.Show(passToMakepkg(dir, args...))
		if err != nil {
			return fmt.Errorf("Error making: %s", base.String())
		}

		pkgdests, version, err := parsePackageList(dir)
		if err != nil {
			return err
		}

		isExplicit := false
		for _, b := range base {
			isExplicit = isExplicit || dp.Explicit.Get(b.Name)
		}
		if config.ReBuild == "no" || (config.ReBuild == "yes" && !isExplicit) {
			for _, split := range base {
				pkgdest, ok := pkgdests[split.Name]
				if !ok {
					return fmt.Errorf("Could not find PKGDEST for: %s", split.Name)
				}

				_, err := os.Stat(pkgdest)
				if os.IsNotExist(err) {
					built = false
				} else if err != nil {
					return err
				}
			}
		} else {
			built = false
		}

		if cmdArgs.existsArg("needed") {
			installed := true
			for _, split := range base {
				if alpmpkg := dp.LocalDB.Pkg(split.Name); alpmpkg == nil || alpmpkg.Version() != version {
					installed = false
				}
			}

			if installed {
				exe.Show(passToMakepkg(dir, "-c", "--nobuild", "--noextract", "--ignorearch"))
				fmt.Println(generic.Cyan(pkg+"-"+version) + generic.Bold(" is up to date -- skipping"))
				continue
			}
		}

		if built {
			exe.Show(passToMakepkg(dir, "-c", "--nobuild", "--noextract", "--ignorearch"))
			fmt.Println(generic.Bold(generic.Yellow(generic.Arrow)),
				generic.Cyan(pkg+"-"+version)+generic.Bold(" already made -- skipping build"))
		} else {
			args := []string{"-cf", "--noconfirm", "--noextract", "--noprepare", "--holdver"}

			if incompatible.Get(pkg) {
				args = append(args, "--ignorearch")
			}

			err := exe.Show(passToMakepkg(dir, args...))
			if err != nil {
				return fmt.Errorf("Error making: %s", base.String())
			}
		}

		arguments := parser.copy()
		arguments.clearTargets()
		arguments.op = "U"
		arguments.delArg("confirm")
		arguments.delArg("noconfirm")
		arguments.delArg("c", "clean")
		arguments.delArg("q", "quiet")
		arguments.delArg("q", "quiet")
		arguments.delArg("y", "refresh")
		arguments.delArg("u", "sysupgrade")
		arguments.delArg("w", "downloadonly")

		oldConfirm := config.NoConfirm

		//conflicts have been checked so answer y for them
		if config.UseAsk {
			ask, _ := strconv.Atoi(cmdArgs.globals["ask"])
			uask := alpm.QuestionType(ask) | alpm.QuestionTypeConflictPkg
			cmdArgs.globals["ask"] = fmt.Sprint(uask)
		} else {
			conflict := false
			for _, split := range base {
				if _, ok := conflicts[split.Name]; ok {
					conflict = true
				}
			}

			if !conflict {
				config.NoConfirm = true
			}
		}

		depArguments := makeArguments()
		depArguments.addArg("D", "asdeps")
		expArguments := makeArguments()
		expArguments.addArg("D", "asexplicit")

		//remotenames: names of all non repo packages on the system
		_, _, localNames, remoteNames, err := filterPackages()
		if err != nil {
			return err
		}

		//cache as a stringset. maybe make it return a string set in the first
		//place
		remoteNamesCache := generic.SliceToStringSet(remoteNames)
		localNamesCache := generic.SliceToStringSet(localNames)

		for _, split := range base {
			pkgdest, ok := pkgdests[split.Name]
			if !ok {
				return fmt.Errorf("Could not find PKGDEST for: %s", split.Name)
			}

			arguments.addTarget(pkgdest)
			if !dp.Explicit.Get(split.Name) && !localNamesCache.Get(split.Name) && !remoteNamesCache.Get(split.Name) {
				depArguments.addTarget(split.Name)
			}

			if dp.Explicit.Get(split.Name) {
				if parser.existsArg("asdeps", "asdep") {
					depArguments.addTarget(split.Name)
				} else if parser.existsArg("asexplicit", "asexp") {
					expArguments.addTarget(split.Name)
				}
			}
		}

		err = exe.Show(passToPacman(arguments))
		if err != nil {
			return err
		}

		var mux sync.Mutex
		var wg sync.WaitGroup
		for _, pkg := range base {
			wg.Add(1)
			go updateVCSData(pkg.Name, srcinfo.Source, &mux, &wg)
		}

		wg.Wait()

		err = saveVCSInfo()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

		if len(depArguments.targets) > 0 {
			_, stderr, err := exe.Capture(passToPacman(depArguments))
			if err != nil {
				return fmt.Errorf("%s%s", stderr, err)
			}
		}
		config.NoConfirm = oldConfirm
	}

	return nil
}
