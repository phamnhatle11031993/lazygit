package gui

import (
	"errors"

	"github.com/fatih/color"
	"github.com/jesseduffield/gocui"
	"github.com/jesseduffield/lazygit/pkg/commands"
)

var (
	// ErrNoCommits : When no commits are found for the branch
	ErrNoCommits = errors.New("No commits for this branch")
)

func (gui *Gui) refreshCommits(g *gocui.Gui) error {
	g.Update(func(*gocui.Gui) error {
		gui.State.Commits = gui.GitCommand.GetCommits()
		v, err := g.View("commits")
		if err != nil {
			panic(err)
		}
		v.Clear()
		red := color.New(color.FgRed)
		yellow := color.New(color.FgYellow)
		white := color.New(color.FgWhite)
		shaColor := white
		for _, commit := range gui.State.Commits {
			if commit.Pushed {
				shaColor = red
			} else {
				shaColor = yellow
			}
			shaColor.Fprint(v, commit.Sha+" ")
			white.Fprintln(v, commit.Name)
		}
		gui.refreshStatus(g)
		if g.CurrentView().Name() == "commits" {
			gui.handleCommitSelect(g, v)
		}
		return nil
	})
	return nil
}

func (gui *Gui) handleResetToCommit(g *gocui.Gui, commitView *gocui.View) error {
	return gui.createConfirmationPanel(g, commitView, "Reset To Commit", "Are you sure you want to reset to this commit?", func(g *gocui.Gui, v *gocui.View) error {
		commit, err := gui.getSelectedCommit(g)
		if err != nil {
			panic(err)
		}
		if output, err := gui.GitCommand.ResetToCommit(commit.Sha); err != nil {
			return gui.createErrorPanel(g, output)
		}
		if err := gui.refreshCommits(g); err != nil {
			panic(err)
		}
		if err := gui.refreshFiles(g); err != nil {
			panic(err)
		}
		gui.resetOrigin(commitView)
		return gui.handleCommitSelect(g, nil)
	}, nil)
}

func (gui *Gui) renderCommitsOptions(g *gocui.Gui) error {
	return gui.renderOptionsMap(g, map[string]string{
		"s":       "squash down",
		"r":       "rename",
		"g":       "reset to this commit",
		"f":       "fixup commit",
		"← → ↑ ↓": "navigate",
	})
}

func (gui *Gui) handleCommitSelect(g *gocui.Gui, v *gocui.View) error {
	if err := gui.renderCommitsOptions(g); err != nil {
		return err
	}
	commit, err := gui.getSelectedCommit(g)
	if err != nil {
		if err != ErrNoCommits {
			return err
		}
		return gui.renderString(g, "main", "No commits for this branch")
	}
	commitText := gui.GitCommand.Show(commit.Sha)
	return gui.renderString(g, "main", commitText)
}

func (gui *Gui) handleCommitSquashDown(g *gocui.Gui, v *gocui.View) error {
	if gui.getItemPosition(v) != 0 {
		return gui.createErrorPanel(g, "Can only squash topmost commit")
	}
	if len(gui.State.Commits) == 1 {
		return gui.createErrorPanel(g, "You have no commits to squash with")
	}
	commit, err := gui.getSelectedCommit(g)
	if err != nil {
		return err
	}
	if output, err := gui.GitCommand.SquashPreviousTwoCommits(commit.Name); err != nil {
		return gui.createErrorPanel(g, output)
	}
	if err := gui.refreshCommits(g); err != nil {
		panic(err)
	}
	gui.refreshStatus(g)
	return gui.handleCommitSelect(g, v)
}

// TODO: move to files panel
func (gui *Gui) anyUnStagedChanges(files []commands.File) bool {
	for _, file := range files {
		if file.Tracked && file.HasUnstagedChanges {
			return true
		}
	}
	return false
}

func (gui *Gui) handleCommitFixup(g *gocui.Gui, v *gocui.View) error {
	if len(gui.State.Commits) == 1 {
		return gui.createErrorPanel(g, "You have no commits to squash with")
	}
	if gui.anyUnStagedChanges(gui.State.Files) {
		return gui.createErrorPanel(g, "Can't fixup while there are unstaged changes")
	}
	branch := gui.State.Branches[0]
	commit, err := gui.getSelectedCommit(g)
	if err != nil {
		return err
	}
	gui.createConfirmationPanel(g, v, "Fixup", "Are you sure you want to fixup this commit? The commit beneath will be squashed up into this one", func(g *gocui.Gui, v *gocui.View) error {
		if output, err := gui.GitCommand.SquashFixupCommit(branch.Name, commit.Sha); err != nil {
			return gui.createErrorPanel(g, output)
		}
		if err := gui.refreshCommits(g); err != nil {
			panic(err)
		}
		return gui.refreshStatus(g)
	}, nil)
	return nil
}

func (gui *Gui) handleRenameCommit(g *gocui.Gui, v *gocui.View) error {
	if gui.getItemPosition(v) != 0 {
		return gui.createErrorPanel(g, "Can only rename topmost commit")
	}
	gui.createPromptPanel(g, v, "Rename Commit", func(g *gocui.Gui, v *gocui.View) error {
		if output, err := gui.GitCommand.RenameCommit(v.Buffer()); err != nil {
			return gui.createErrorPanel(g, output)
		}
		if err := gui.refreshCommits(g); err != nil {
			panic(err)
		}
		return gui.handleCommitSelect(g, v)
	})
	return nil
}

func (gui *Gui) getSelectedCommit(g *gocui.Gui) (commands.Commit, error) {
	v, err := g.View("commits")
	if err != nil {
		panic(err)
	}
	if len(gui.State.Commits) == 0 {
		return commands.Commit{}, ErrNoCommits
	}
	lineNumber := gui.getItemPosition(v)
	if lineNumber > len(gui.State.Commits)-1 {
		gui.Log.Info("potential error in getSelected Commit (mismatched ui and state)", gui.State.Commits, lineNumber)
		return gui.State.Commits[len(gui.State.Commits)-1], nil
	}
	return gui.State.Commits[lineNumber], nil
}
