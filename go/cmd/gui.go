package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sonar-solutions/sonar-migration-tool/internal/gui"
	"github.com/sonar-solutions/sonar-migration-tool/internal/wizard"
	"github.com/spf13/cobra"
)

var guiCmd = &cobra.Command{
	Use:   "gui",
	Short: "Launch browser-based GUI for the migration wizard",
	Long:  "Starts a local HTTP server and opens the migration wizard in your default browser.",
	RunE:  runGUI,
}

func init() {
	guiCmd.Flags().String("export_directory", "/app/files/", "Root directory to output the export")
	guiCmd.Flags().String("addr", "localhost:0", "Address to bind the HTTP server (default: random port)")
	guiCmd.Flags().Bool("no-browser", false, "Do not open the browser automatically")
}

func runGUI(cmd *cobra.Command, args []string) error {
	exportDir, _ := cmd.Flags().GetString("export_directory")
	addr, _ := cmd.Flags().GetString("addr")
	noBrowser, _ := cmd.Flags().GetBool("no-browser")

	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create hub with a nil prompter initially (set when wizard starts).
	hub := gui.NewHub(nil)

	// Frontend filesystem: strip the "frontend" prefix from embedded FS.
	frontend, err := fs.Sub(gui.FrontendDist, "frontend")
	if err != nil {
		return fmt.Errorf("embedded frontend: %w", err)
	}

	srv := gui.NewServer(addr, exportDir, hub, frontend)

	// Wizard lifecycle.
	var (
		wizMu     sync.Mutex
		wizCancel context.CancelFunc
		wizActive bool
	)

	hub.OnStartWizard = func() {
		wizMu.Lock()
		defer wizMu.Unlock()
		if wizActive {
			hub.Send(gui.ServerMessage{
				Type:    gui.TypeDisplayWarning,
				Message: "Wizard is already running.",
			})
			return
		}
		wizActive = true

		wizCtx, wCancel := context.WithCancel(ctx)
		wizCancel = wCancel

		prompter := gui.NewWebPrompter(wizCtx, hub.Send)
		hub.SetPrompter(prompter)
		hub.Send(gui.ServerMessage{Type: gui.TypeWizardStarted})

		go func() {
			err := wizard.Run(wizCtx, prompter, exportDir)

			wizMu.Lock()
			wizActive = false
			wizMu.Unlock()

			msg := gui.ServerMessage{Type: gui.TypeWizardFinished}
			if err != nil && wizCtx.Err() == nil {
				errStr := err.Error()
				msg.Error = &errStr
			}
			hub.Send(msg)
		}()
	}

	hub.OnCancelWizard = func() {
		wizMu.Lock()
		defer wizMu.Unlock()
		if wizCancel != nil {
			wizCancel()
		}
	}

	// Start server in a goroutine so we can open the browser after binding.
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	// Wait for the listener to bind, then open browser.
	if !noBrowser {
		go func() {
			select {
			case <-srv.Ready():
				u := srv.URL()
				log.Printf("gui: opening browser at %s", u)
				if err := gui.OpenBrowser(u); err != nil {
					log.Printf("gui: could not open browser: %v (open %s manually)", err, u)
				}
			case <-ctx.Done():
			}
		}()
	}

	return <-errCh
}
