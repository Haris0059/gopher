// Source: src/cli/handlers/util.tsx — doctorHandler
package handlers

import (
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/Haris0059/gopher/pkg/doctor"
	"github.com/Haris0059/gopher/pkg/ui/screens"
	"golang.org/x/term"
)

// DoctorOpts configures the doctor handler.
type DoctorOpts struct {
	Output  io.Writer // defaults to os.Stdout
	Version string    // current version; "" leaves doctor.Version as-is

	// Run renders the collected diagnostics. Nil uses DefaultRunDoctor, which
	// launches the full-screen Doctor TUI when Output is a terminal and
	// otherwise prints the same sections as plain text.
	Run func(w io.Writer, data *doctor.DiagnosticData) error
}

// Doctor handles `claude doctor`.
// It launches the diagnostics screen (Doctor TUI component).
// Source: src/cli/handlers/util.tsx — doctorHandler
func Doctor(opts DoctorOpts) int {
	w := output(opts.Output)

	// Analytics: tengu_doctor_command (stub — T-analytics)

	if opts.Version != "" {
		doctor.Version = opts.Version
	}

	installType := DefaultDetectInstallType()
	data := doctor.Collect(doctor.CollectOptions{
		InstallationType:    string(installType),
		PackageManager:      string(DefaultDetectPackageManager()),
		ConfigInstallMethod: string(NormalizeInstallType(installType)),
		FetchDistTags:       doctor.DefaultFetchDistTags,
	})

	run := opts.Run
	if run == nil {
		run = DefaultRunDoctor
	}

	if err := run(w, data); err != nil {
		fmt.Fprintf(w, "doctor failed: %v\n", err)
		return 1
	}
	return 0
}

// DefaultRunDoctor renders data as the full-screen Doctor TUI when w is a
// terminal, and as plain text otherwise (piped output, CI, tests).
// Source: src/cli/handlers/util.tsx — doctorHandler renders <Doctor/> via Ink;
// the plain-text fallback has no TS equivalent (Ink always has a TTY) and
// exists so `gopher doctor` behaves sensibly when piped.
func DefaultRunDoctor(w io.Writer, data *doctor.DiagnosticData) error {
	cfg := screens.NewDoctorConfigFromDiagnostic(data)

	if f, ok := w.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		p := tea.NewProgram(newDoctorProgram(cfg))
		_, err := p.Run()
		return err
	}

	fmt.Fprintln(w, screens.RenderDoctorText(cfg))
	return nil
}

// doctorProgram wraps screens.DoctorModel for standalone use: the screen
// emits screens.DoctorDoneMsg expecting a parent app to tear it down, so a
// bare tea.Program has to translate that into tea.Quit itself.
type doctorProgram struct {
	*screens.DoctorModel
}

func newDoctorProgram(cfg screens.DoctorConfig) *doctorProgram {
	return &doctorProgram{DoctorModel: screens.NewDoctorModel(cfg)}
}

func (p *doctorProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(screens.DoctorDoneMsg); ok {
		return p, tea.Quit
	}
	m, cmd := p.DoctorModel.Update(msg)
	p.DoctorModel = m.(*screens.DoctorModel)
	return p, cmd
}
