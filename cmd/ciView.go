package cmd

import (
	"fmt"
	"log"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/pkg/errors"
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
		a := tview.NewApplication()
		defer recoverPanic(a)
		var (
			remote string
		)

		branch, err := git.CurrentBranch()
		if err != nil {
			log.Fatal(err)
		}

		remote = determineSourceRemote(branch)
		if len(args) > 0 {
			ok, err := git.IsRemote(args[0])
			if err != nil || !ok {
				log.Fatal(args[0], "is not a remote:", err)
			}
			remote = args[0]
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
		root := tview.NewPages()
		root.SetBorderPadding(1, 1, 2, 2)

		boxes = make(map[string]*tview.TextView)
		jobsCh := make(chan []*gitlab.Job)

		a.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
				a.Stop()
				return nil
			}
			return event
		})
		go updateJobs(a, jobsCh, project.ID, branch)
		go refreshScreen(a)
		if err := a.SetRoot(root, true).SetBeforeDrawFunc(jobsView(a, jobsCh, root)).SetAfterDrawFunc(connectJobsView(a)).Run(); err != nil {
			log.Fatal(err)
		}
	},
}

var (
	jobs  []*gitlab.Job
	boxes map[string]*tview.TextView
)

func jobsView(app *tview.Application, jobsCh chan []*gitlab.Job, root *tview.Pages) func(screen tcell.Screen) bool {
	return func(screen tcell.Screen) bool {
		defer recoverPanic(app)
		screen.Clear()
		select {
		case jobs = <-jobsCh:
		default:
			if len(jobs) == 0 {
				jobs = <-jobsCh
			}
		}
		px, _, maxX, maxY := root.GetInnerRect()
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
			// retryChar := '⟳'
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

func recoverPanic(app *tview.Application) {
	if r := recover(); r != nil {
		app.Stop()
		log.Fatalf("%s\n%s\n", r, string(debug.Stack()))
	}
}

func refreshScreen(app *tview.Application) {
	defer recoverPanic(app)
	for {
		app.Draw()
		time.Sleep(time.Second * 1)
	}
}

func updateJobs(app *tview.Application, jobsCh chan []*gitlab.Job, pid interface{}, branch string) {
	defer recoverPanic(app)
	for {
		jobs, err := lab.CIJobs(pid, branch)
		if len(jobs) == 0 || err != nil {
			app.Stop()
			log.Fatal(errors.Wrap(err, "failed to find ci jobs"))
		}
		jobsCh <- latestJobs(jobs)
		time.Sleep(time.Second * 5)
	}
}

func connectJobsView(app *tview.Application) func(screen tcell.Screen) {
	return func(screen tcell.Screen) {
		defer recoverPanic(app)
		err := connectJobs(screen, jobs, boxes)
		if err != nil {
			app.Stop()
			log.Fatal(err)
		}
	}
}

func connectJobs(screen tcell.Screen, jobs []*gitlab.Job, boxes map[string]*tview.TextView) error {
	for i, j := range jobs {
		if _, ok := boxes["jobs-"+j.Name]; !ok {
			return errors.Errorf("jobs-%s not found at index: %d", jobs[i].Name, i)
		}
	}
	var padding int
	// find the abount of space between two jobs is adjacent stages
	for i, k := 0, 1; k < len(jobs); i, k = i+1, k+1 {
		if jobs[i].Stage == jobs[k].Stage {
			continue
		}
		x1, _, w, _ := boxes["jobs-"+jobs[i].Name].GetRect()
		x2, _, _, _ := boxes["jobs-"+jobs[k].Name].GetRect()
		stageWidth := x2 - x1 - w
		switch {
		case stageWidth <= 3:
			padding = 1
		case stageWidth <= 6:
			padding = 2
		case stageWidth > 6:
			padding = 3
		}
	}
	for i, k := 0, 1; k < len(jobs); i, k = i+1, k+1 {
		v1 := boxes["jobs-"+jobs[i].Name]
		v2 := boxes["jobs-"+jobs[k].Name]
		connect(screen, v1.Box, v2.Box, padding,
			jobs[i].Stage == jobs[0].Stage,           // is first stage?
			jobs[i].Stage == jobs[len(jobs)-1].Stage) // is last stage?
	}
	return nil
}

func connect(screen tcell.Screen, v1 *tview.Box, v2 *tview.Box, padding int, firstStage, lastStage bool) {
	x1, y1, w, h := v1.GetRect()
	x2, y2, _, _ := v2.GetRect()

	dx, dy := x2-x1, y2-y1

	p := padding

	// drawing stages
	if dx != 0 {
		hline(screen, x1+w, y2+h/2, dx-w)
		if dy != 0 {
			// dy != 0 means the last stage had multple jobs
			screen.SetContent(x1+w+p-1, y2+h/2, '┳', nil, tcell.StyleDefault)
		}
		return
	}

	// Drawing a job in the same stage
	// left of view
	if !firstStage {
		if r, _, _, _ := screen.GetContent(x2-p, y1+h/2); r == '┗' {
			screen.SetContent(x2-p, y1+h/2, '┣', nil, tcell.StyleDefault)
		} else {
			screen.SetContent(x2-p, y1+h/2, '┳', nil, tcell.StyleDefault)
		}

		for i := 1; i < p; i++ {
			screen.SetContent(x2-i, y2+h/2, '━', nil, tcell.StyleDefault)
		}
		screen.SetContent(x2-p, y2+h/2, '┗', nil, tcell.StyleDefault)

		vline(screen, x2-p, y1+h-1, dy-1)
	}
	// right of view
	if !lastStage {
		if r, _, _, _ := screen.GetContent(x2+w+p-1, y1+h/2); r == '┛' {
			screen.SetContent(x2+w+p-1, y1+h/2, '┫', nil, tcell.StyleDefault)
		}
		for i := 0; i < p-1; i++ {
			screen.SetContent(x2+w+i, y2+h/2, '━', nil, tcell.StyleDefault)
		}
		screen.SetContent(x2+w+p-1, y2+h/2, '┛', nil, tcell.StyleDefault)

		vline(screen, x2+w+p-1, y1+h-1, dy-1)
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

// latestJobs returns a list of unique jobs favoring the last stage+name
// version of a job in the provided list
func latestJobs(jobs []*gitlab.Job) []*gitlab.Job {
	var (
		lastJob = make(map[string]*gitlab.Job, len(jobs))
		dupIdx  = -1
	)
	for i, j := range jobs {
		_, ok := lastJob[j.Stage+j.Name]
		if dupIdx == -1 && ok {
			dupIdx = i
		}
		// always want the latest job
		lastJob[j.Stage+j.Name] = j
	}
	if dupIdx == -1 {
		dupIdx = len(jobs)
	}
	// first duplicate marks where retries begin
	outJobs := make([]*gitlab.Job, dupIdx)
	for i := range outJobs {
		j := jobs[i]
		outJobs[i] = lastJob[j.Stage+j.Name]
	}

	return outJobs
}

func init() {
	ciCmd.AddCommand(ciViewCmd)
}
