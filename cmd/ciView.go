package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"

	"github.com/xanzy/go-gitlab"

	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// ciViewCmd represents the ci command
var ciViewCmd = &cobra.Command{
	Use:   "view [remote]",
	Short: "(beta) render the CI Pipeline to the terminal",
	Long: `This feature is currently under development and only supports
viewing jobs. In the future we hope to support starting jobs and jumping to
logs.

Feedback Welcome!: https://github.com/zaquestion/lab/issues/74`,
	Run: func(cmd *cobra.Command, args []string) {
		remote, _, err := parseArgsRemote(args)
		if err != nil {
			log.Fatal(err)
		}
		if remote == "" {
			remote = forkedFromRemote
		}

		// See if we're in a git repo or if global is set to determine
		// if this should be a personal snippet
		rn, err := git.PathWithNameSpace(remote)
		if err != nil {
			log.Fatal(err)
		}
		project, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}
		sha, err := git.Sha("HEAD")
		if err != nil {
			log.Fatal(err)
		}
		root := tview.NewPages()
		root.SetBorderPadding(1, 1, 2, 14)

		boxes = make(map[string]*tview.TextView)
		jobsCh := make(chan []gitlab.Job)
		a := tview.NewApplication()
		go updateJobs(a, jobsCh, project.ID, sha)
		if err := a.SetRoot(root, true).SetBeforeDrawFunc(jobsView(jobsCh, root)).SetAfterDrawFunc(connectJobs).Run(); err != nil {
			log.Fatal(err)
		}
	},
}

var (
	boxes map[string]*tview.TextView
)

func jobsView(jobsCh chan []gitlab.Job, root *tview.Pages) func(screen tcell.Screen) bool {
	return func(screen tcell.Screen) bool {
		screen.Clear()
		select {
		case jobs = <-jobsCh:
		default:
			if len(jobs) == 0 {
				jobs = <-jobsCh
			}
		}
		px, _, maxX, maxY := root.GetInnerRect()
		//px, py, maxX, maxY := root.GetInnerRect()
		//fmt.Printf("root x: %d, y: %d, w: %d, h: %d\n", px, py, maxX, maxY)
		var (
			stages    = 0
			lastStage = ""
		)
		// get the number of stages
		for _, j := range jobs {
			if j.Stage != lastStage {
				lastStage = j.Stage
				stages++
			}
		}
		lastStage = ""
		var (
			rowIdx   = 0
			stageIdx = 0
			maxTitle = 20
		)
		for _, j := range jobs {
			boxX := px + (maxX / stages * stageIdx)
			if j.Stage != lastStage {
				rowIdx = 0
				stageIdx++
				lastStage = j.Stage
				key := "stage-" + j.Stage

				x, y, w, h := boxX, maxY/6-4, maxTitle+2, 3
				b := box(root, key, x, y, w, h)
				b.SetText(strings.Title(j.Stage))
				b.SetTextAlign(tview.AlignCenter)

			}
		}
		lastStage = jobs[0].Stage
		rowIdx = 0
		stageIdx = 0
		for _, j := range jobs {
			if j.Stage != lastStage {
				rowIdx = 0
				lastStage = j.Stage
				stageIdx++
			}
			//fmt.Printf("\nstage: %s, stageIdx: %d, rowIdx: %d\n", j.Stage, stageIdx, rowIdx)
			boxX := px + (maxX / stages * stageIdx)

			key := "jobs-" + j.Name
			x, y, w, h := boxX, maxY/6+(rowIdx*5), maxTitle+2, 4
			b := box(root, key, x, y, w, h)
			b.SetTitle(j.Name)
			// The scope of jobs to show, one or array of: created, pending, running,
			// failed, success, canceled, skipped; showing all jobs if none provided
			var statChar rune
			switch j.Status {
			case "success":
				b.SetBorderColor(tcell.ColorGreen)
				statChar = '✔'
			case "failed":
				b.SetBorderColor(tcell.ColorRed)
				statChar = '✘'
			case "running":
				b.SetBorderColor(tcell.ColorBlue)
				statChar = '●'
			case "pending":
				b.SetBorderColor(tcell.ColorYellow)
				statChar = '●'
			case "manual":
				b.SetBorderColor(tcell.ColorGrey)
				statChar = '●'
			}
			retryChar := '⟳'
			_ = retryChar
			title := fmt.Sprintf("%c %s", statChar, j.Name)
			// trim the suffix if it matches the stage, I've seen
			// the pattern in 2 different places to handle
			// different stages for the same service and it tends
			// to make the title spill over the max
			title = strings.TrimSuffix(title, ":"+j.Stage)
			b.SetTitle(title)
			// tview default aligns center, which is nice, but if
			// the title is too long we want to bias towards seeing
			// the beginning of it
			if tview.StringWidth(title) > maxTitle {
				b.SetTitleAlign(tview.AlignLeft)
			}
			if j.StartedAt != nil {
				end := time.Now()
				if j.FinishedAt != nil {
					end = *j.FinishedAt
				}
				b.SetText("\n" + fmtDuration(end.Sub(*j.StartedAt)))
				b.SetTextAlign(tview.AlignRight)
			}
			rowIdx++

		}
		return false
	}
}
func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02dm %02ds", m, s)
}
func box(root *tview.Pages, key string, x, y, w, h int) *tview.TextView {
	//fmt.Printf("key: %s, x: %d, y: %d, w: %d, h: %d\n", key, x, y, w, h)
	b, ok := boxes[key]
	if !ok {
		b = tview.NewTextView()
		b.SetBorder(true)
		boxes[key] = b
	}
	b.SetRect(x, y, w, h)
	root.AddPage(key, b, false, true)
	return b
}

func updateJobs(app *tview.Application, jobsCh chan []gitlab.Job, pid interface{}, sha string) {
	for {
		jobs, err := lab.CIJobs(pid, sha)
		if err != nil {
			app.Stop()
			log.Fatal(err)
		}
		go app.Draw()
		jobsCh <- jobs
		time.Sleep(time.Second * 5)
	}
}

var jobs []gitlab.Job

func connectJobs(screen tcell.Screen) {
	for i, k := 0, 1; k < len(jobs); i, k = i+1, k+1 {
		v1, ok := boxes["jobs-"+jobs[i].Name]
		if !ok {
			log.Fatal("not okay")
		}
		v2, ok := boxes["jobs-"+jobs[k].Name]
		if !ok {
			log.Fatal("not okay")
		}
		connect(screen, v1.Box, v2.Box, jobs[i].Stage == jobs[0].Stage, jobs[i].Stage == jobs[len(jobs)-1].Stage)
	}
}

func connect(screen tcell.Screen, v1 *tview.Box, v2 *tview.Box, firstStage, lastStage bool) {
	x1, y1, w, h := v1.GetRect()
	x2, y2, _, _ := v2.GetRect()

	dx, dy := x2-x1, y2-y1

	// dy != 0 means the last stage had multple jobs
	if dy != 0 && dx != 0 {
		hline(screen, x1+w, y2+h/2, dx-w)
		screen.SetContent(x1+w+2, y2+h/2, '┳', nil, tcell.StyleDefault)
		return
	}
	if dy == 0 {
		hline(screen, x1+w, y1+h/2, dx-w)
		return
	}

	// cells := screen.CellBuffer()
	// tw, _ := screen.Size()

	// '┣' '┫'
	// TODO: fix drawing the last stage (don't draw right side of box)
	// TODO: fix drawing the first stage (don't draw left side of box)

	// Drawing a job in the same stage
	// left of view
	if !firstStage {
		if r, _, _, _ := screen.GetContent(x2-3, y1+h/2); r == '┗' {
			screen.SetContent(x2-3, y1+h/2, '┣', nil, tcell.StyleDefault)
		} else {
			screen.SetContent(x2-3, y1+h/2, '┳', nil, tcell.StyleDefault)
		}

		screen.SetContent(x2-1, y2+h/2, '━', nil, tcell.StyleDefault)
		screen.SetContent(x2-2, y2+h/2, '━', nil, tcell.StyleDefault)
		screen.SetContent(x2-3, y2+h/2, '┗', nil, tcell.StyleDefault)

		// NOTE: unsure what the 2nd arg (y), "-1" is needed for. Maybe due to
		// padding? This showed up after migrating from termbox
		vline(screen, x2-3, y1+h-1, dy-1)
	}
	// right of view
	if !lastStage {
		vline(screen, x2+w+2, y1+h-1, dy-1)

		if r, _, _, _ := screen.GetContent(x2+w+2, y1+h/2); r == '┛' {
			screen.SetContent(x2+w+2, y1+h/2, '┫', nil, tcell.StyleDefault)
		}
		screen.SetContent(x2+w, y2+h/2, '━', nil, tcell.StyleDefault)
		screen.SetContent(x2+w+1, y2+h/2, '━', nil, tcell.StyleDefault)
		screen.SetContent(x2+w+2, y2+h/2, '┛', nil, tcell.StyleDefault)
	}

}

func hline(screen tcell.Screen, x, y, l int) {
	for i := 0; i < l; i++ {
		screen.SetContent(x+i, y, '━', nil, tcell.StyleDefault)
	}
}

func vline(screen tcell.Screen, x, y, l int) {
	for i := 0; i < l; i++ {
		screen.SetContent(x, y+i, '┃', nil, tcell.StyleDefault)
	}
}

func init() {
	ciCmd.AddCommand(ciViewCmd)
}
