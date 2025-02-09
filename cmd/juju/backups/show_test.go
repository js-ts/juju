// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backups_test

import (
	"github.com/juju/cmd/v3"
	"github.com/juju/cmd/v3/cmdtesting"
	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/cmd/juju/backups"
)

type showSuite struct {
	BaseBackupsSuite
	subcommand cmd.Command
}

var _ = gc.Suite(&showSuite{})

func (s *showSuite) SetUpTest(c *gc.C) {
	s.BaseBackupsSuite.SetUpTest(c)
	s.subcommand = backups.NewShowCommandForTest(s.store)
}

func (s *showSuite) TestOkay(c *gc.C) {
	s.setSuccess()
	ctx, err := cmdtesting.RunCommand(c, s.subcommand, s.metaresult.ID)
	c.Check(err, jc.ErrorIsNil)

	c.Check(cmdtesting.Stderr(ctx), gc.Equals, "")
	c.Check(cmdtesting.Stdout(ctx), gc.Equals, MetaResultString)
}

func (s *showSuite) TestError(c *gc.C) {
	s.setFailure("failed!")
	_, err := cmdtesting.RunCommand(c, s.subcommand, s.metaresult.ID)
	c.Check(errors.Cause(err), gc.ErrorMatches, "failed!")
}
