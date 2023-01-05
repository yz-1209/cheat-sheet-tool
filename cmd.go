package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "embed"
)

//go:embed version.txt
var version string

type CmdKind int

const (
	CmdHelp CmdKind = iota
	CmdVersion
	CmdFind
	CmdEdit
	CmdUpdate
)

func (c CmdKind) String() string {
	return []string{"help", "version", "find", "edit", "update"}[c]
}

func CreateCommand(fs *flag.FlagSet) *Command {
	withLog := func() CmdOption {
		logFlag := fs.Lookup(LogFlag)
		if logFlag.Value.String() != "false" {
			return WithFlag(LogFlag, "true")
		}

		return func(c *Command) {}
	}

	helpFlag := fs.Lookup(HelpFlag)
	if helpFlag.Value.String() == "true" {
		return NewCommand(CmdHelp, withLog())
	}

	verFlag := fs.Lookup(VerFlag)
	if verFlag.Value.String() == "true" {
		return NewCommand(CmdVersion, withLog())
	}

	updateFlag := fs.Lookup(UpdateFlag)
	if updateFlag.Value.String() == "true" {
		return NewCommand(CmdUpdate, withLog())
	}

	editFlag := fs.Lookup(EditFlag)
	if val := editFlag.Value.String(); val != "" {
		args := append([]string{val}, fs.Args()...)
		return NewCommand(CmdEdit, WithArgs(args), withLog())
	}

	return NewCommand(CmdFind, WithArgs(fs.Args()), withLog())
}

type CmdOption func(*Command)

func WithArgs(args []string) CmdOption {
	return func(c *Command) {
		c.Args = args
	}
}

func WithFlag(name, val string) CmdOption {
	return func(c *Command) {
		c.Flags[name] = val
	}
}

func NewCommand(kind CmdKind, options ...CmdOption) *Command {
	cmd := &Command{
		Cmd:   kind,
		Flags: make(map[string]string),
	}

	for _, o := range options {
		o(cmd)
	}

	return cmd
}

type Command struct {
	Cmd   CmdKind
	Args  []string
	Flags map[string]string
}

func (c *Command) PrintLog() bool {
	_, ok := c.Flags[LogFlag]
	return ok
}

func (c *Command) Filename() string {
	return strings.Join(c.Args, "-") + ".md"
}

func DefaultConfig() (*Config, error) {
	dirname, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cheatSheetDir := filepath.Join(dirname, ".cheat-sheet")
	_, err = os.Stat(cheatSheetDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}

		if err = os.Mkdir(cheatSheetDir, 0755); err != nil {
			return nil, err
		}
	}

	tldrCachePath := filepath.Join(dirname, ".tldr/cache/pages")
	return &Config{
		CheatSheetsDir: cheatSheetDir,
		TldrPath:       "tldr",
		TldrCachePath:  tldrCachePath,
		TldrPages:      []string{"common", "linux"},
		EditorPath:     "vim",
	}, nil
}

type Config struct {
	CheatSheetsDir string
	TldrPath       string
	TldrCachePath  string
	TldrPages      []string
	EditorPath     string
}

func NewTldr(cmdPath, cachePath string, pages []string) *Tldr {
	return &Tldr{
		CmdPath:   cmdPath,
		CachePath: cachePath,
		pages:     pages,
	}
}

type Tldr struct {
	CmdPath   string
	CachePath string
	pages     []string
}

func (t *Tldr) run(args ...string) error {
	cmd := exec.Command(t.CmdPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		// If cheat-sheet not found, tldr exits with code 3.
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			if exitCode == 3 {
				return nil
			}
		}

		return err
	}
	return nil
}

func (t *Tldr) Find(args ...string) error {
	return t.run(args...)
}

func (t *Tldr) Render(path string) error {
	args := []string{"--render", path}
	return t.run(args...)
}

func (t *Tldr) Update() error {
	return t.run("--update")
}

func (t *Tldr) FindFileInCache(filename string) (string, error) {
	var dirs []string
	for _, page := range t.pages {
		dirs = append(dirs, filepath.Join(t.CachePath, page))
	}

	for _, dir := range dirs {
		ok, err := IsFileExists(dir, filename)
		if err != nil {
			return "", err
		}

		if ok {
			return dir, nil
		}
	}

	return "", nil
}

func (t *Tldr) Version() (string, error) {
	cmd := exec.Command(t.CmdPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), err
}

func NewExecutor(cfg *Config) *Executor {
	return &Executor{
		cfg:  cfg,
		tldr: NewTldr(cfg.TldrPath, cfg.TldrCachePath, cfg.TldrPages),
	}
}

type Executor struct {
	cfg  *Config
	tldr *Tldr
}

func (e *Executor) Exec(cmd *Command) error {
	var err error
	switch cmd.Cmd {
	case CmdHelp:
		e.PrintHelp()
	case CmdVersion:
		err = e.PrintVersion()
	case CmdFind:
		err = e.Find(cmd)
	case CmdUpdate:
		err = e.Update(cmd)
	case CmdEdit:
		err = e.Edit(cmd)
	default:
		err = fmt.Errorf("unrecognized command: '%v' \n", cmd.Cmd)
	}

	return err
}

func (e *Executor) PrintHelp() {
	fmt.Println("Usage: cs command [options]")
	fmt.Println("Examples:")
	fmt.Printf("\tTo list cheat-sheet of `git`\n")
	fmt.Printf("\t$ cs git\n")
	fmt.Println()
	fmt.Printf("\tTo edit cheat-sheet of `git`\n")
	fmt.Printf("\t$ cs -e git\n")
}

func (e *Executor) PrintVersion() error {
	tldrVersion, err := e.tldr.Version()
	if err != nil {
		return err
	}

	fmt.Printf("cheat-sheet:\t%v\n", version)
	fmt.Printf("tldr:\t%v\n", tldrVersion)
	return nil
}

func (e *Executor) Find(cmd *Command) error {
	ok, err := IsFileExists(e.cfg.CheatSheetsDir, cmd.Filename())
	if err != nil {
		return err
	}

	if cmd.PrintLog() {
		log.Printf("has found local cheat-sheet: %v\n", ok)
	}

	if ok {
		return e.tldr.Render(filepath.Join(e.cfg.CheatSheetsDir, cmd.Filename()))
	}

	return e.tldr.Find(cmd.Args...)
}

func (e *Executor) Edit(cmd *Command) error {
	ok, err := IsFileExists(e.cfg.CheatSheetsDir, cmd.Filename())
	if err != nil {
		return err
	}

	if ok {
		return e.editLocalCheatSheet(cmd)
	}

	dirname, err := e.tldr.FindFileInCache(cmd.Filename())
	if err != nil {
		return err
	}

	if cmd.PrintLog() {
		log.Printf("find cheat sheet stored in '%v' of tldr cache\n", dirname)
	}

	if dirname != "" {
		src := filepath.Join(dirname, cmd.Filename())
		dest := filepath.Join(e.cfg.CheatSheetsDir, cmd.Filename())
		if err := CopyFile(src, dest); err != nil {
			return err
		}
	}

	return e.editLocalCheatSheet(cmd)
}

func (e *Executor) editLocalCheatSheet(cmd *Command) error {
	cheatSheetFilePath := filepath.Join(e.cfg.CheatSheetsDir, cmd.Filename())
	editCmd := exec.Command(e.cfg.EditorPath, cheatSheetFilePath)
	editCmd.Stdin = os.Stdin
	editCmd.Stdout = os.Stdout
	editCmd.Stderr = os.Stderr

	return editCmd.Run()
}

func (e *Executor) Update(cmd *Command) error {
	return e.tldr.Update()
}

func IsFileExists(dirname, filename string) (bool, error) {
	_, err := os.Stat(filepath.Join(dirname, filename))
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

func CopyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}
