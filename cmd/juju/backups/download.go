// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backups

import (
	"fmt"
	"io"

	"github.com/juju/cmd/v3"
	"github.com/juju/errors"
	"github.com/juju/gnuflag"

	jujucmd "github.com/juju/juju/cmd"
	"github.com/juju/juju/cmd/modelcmd"
	"github.com/juju/juju/state/backups"
)

const downloadDoc = `
download-backup retrieves a backup archive file.

If --filename is not used, the archive is downloaded to a temporary
location and the filename is printed to stdout.
`

// NewDownloadCommand returns a commant used to download backups.
func NewDownloadCommand() cmd.Command {
	return modelcmd.Wrap(&downloadCommand{})
}

// downloadCommand is the sub-command for downloading a backup archive.
type downloadCommand struct {
	CommandBase
	// Filename is where to save the downloaded archive.
	Filename string
	// ID is the backup ID to download.
	ID string
}

// Info implements Command.Info.
func (c *downloadCommand) Info() *cmd.Info {
	return jujucmd.Info(&cmd.Info{
		Name:    "download-backup",
		Args:    "<ID>",
		Purpose: "Get an archive file.",
		Doc:     downloadDoc,
	})
}

// SetFlags implements Command.SetFlags.
func (c *downloadCommand) SetFlags(f *gnuflag.FlagSet) {
	c.CommandBase.SetFlags(f)
	f.StringVar(&c.Filename, "filename", "", "Download target")
}

// Init implements Command.Init.
func (c *downloadCommand) Init(args []string) error {
	if len(args) == 0 {
		return errors.New("missing ID")
	}
	id, args := args[0], args[1:]
	if err := cmd.CheckEmpty(args); err != nil {
		return errors.Trace(err)
	}
	c.ID = id
	return nil
}

// Run implements Command.Run.
func (c *downloadCommand) Run(ctx *cmd.Context) error {
	if err := c.validateIaasController(c.Info().Name); err != nil {
		return errors.Trace(err)
	}
	client, err := c.NewAPIClient()
	if err != nil {
		return errors.Trace(err)
	}
	defer client.Close()

	// Download the archive.
	resultArchive, err := client.Download(c.ID)
	if err != nil {
		return errors.Trace(err)
	}
	defer resultArchive.Close()

	// Prepare the local archive.
	filename := c.ResolveFilename()
	archive, err := c.Filesystem().Create(filename)
	if err != nil {
		return errors.Annotate(err, "while creating local archive file")
	}
	defer archive.Close()

	// Write out the archive.
	_, err = io.Copy(archive, resultArchive)
	if err != nil {
		return errors.Annotate(err, "while copying local archive file")
	}

	// Print the local filename.
	fmt.Fprintln(ctx.Stdout, filename)
	return nil
}

// ResolveFilename returns the filename used by the command.
func (c *downloadCommand) ResolveFilename() string {
	filename := c.Filename
	if filename == "" {
		filename = backups.FilenamePrefix + c.ID + ".tar.gz"
	}
	return filename
}
