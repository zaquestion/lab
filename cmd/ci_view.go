package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/gdamore/tcell/v2"
	"github.com/pkg/errors"
	"github.com/rivo/tview"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"

	"github.com/lunixbochs/vtclean"
	gitlab "github.com/xanzy/go-gitlab"

	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	projectID  int
	pipelineID int
)

// ciViewCmd represents the ci command
var ciViewCmd = &cobra.Command{
	Use:   "view [remote [branch/tag]]",
	Short: "View, run, trace, and/or cancel CI jobs current pipeline",
	Long: heredoc.Doc(`
		Supports viewing, running, tracing, and canceling jobs.

		The <tab> key navigates and <enter> confirms. Also supports vi style
		(hjkl,Gg) bindings and arrow keys for navigating jobs and logs.

		'r', 'p' to run/retry/play a job
		't' to toggle trace/logs (runs in background)
		'T' to toggle trace/logs (suspending application)
		'c' to cancel job`),
	Example: heredoc.Doc(`
		lab ci view
		lab ci view upstream --merge-request`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		a := tview.NewApplication()
		defer recoverPanic(a)
		var (
			rn  string
			err error
		)

		forMR, err := cmd.Flags().GetBool("merge-request")
		if err != nil {
			log.Fatal(err)
		}

		followBridge, err = cmd.Flags().GetBool("follow")
		if err != nil {
			log.Fatal(err)
		}

		rn, pipelineID, err = getPipelineFromArgs(args, forMR)
		if err != nil {
			log.Fatal(err)
		}

		project, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}
		projectID = project.ID
		root := tview.NewPages()
		root.SetBorderPadding(1, 1, 2, 2)

		boxes = make(map[string]*tview.TextView)
		jobsCh := make(chan []*gitlab.Job)
		inputCh := make(chan struct{}, 1)

		screen, err := tcell.NewScreen()
		if err != nil {
			log.Fatal(err)
		}
		screen.Init()

		var navi navigator
		a.SetInputCapture(inputCapture(a, root, navi, inputCh))
		go updateJobs(a, jobsCh)
		go func() {
			defer recoverPanic(a)
			for {
				a.SetFocus(root)
				jobsView(a, jobsCh, inputCh, root)
				a.Draw()
			}
		}()
		if err := a.SetScreen(screen).SetRoot(root, true).SetAfterDrawFunc(connectJobsView(a)).Run(); err != nil {
			log.Fatal(err)
		}
	},
}

func inputCapture(a *tview.Application, root *tview.Pages, navi navigator, inputCh chan struct{}) func(event *tcell.EventKey) *tcell.EventKey {
	queueInputUpdate := func() {
		if len(inputCh) == 0 {
			inputCh <- struct{}{}
		}
	}
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' || event.Key() == tcell.KeyEscape {
			switch {
			case modalVisible:
				modalVisible = !modalVisible
				root.HidePage("yesno")
				queueInputUpdate()
			case logsVisible:
				logsVisible = !logsVisible
				root.HidePage("logs-" + curJob.Name)
				queueInputUpdate()
				a.ForceDraw()
			default:
				a.Stop()
				return nil
			}
		}
		if !modalVisible && !logsVisible {
			curJob = navi.Navigate(jobs, event)
			root.SendToFront("jobs-" + curJob.Name)
			// update jobs view on input changes
			queueInputUpdate()
		}
		switch event.Rune() {
		case 'c':
			job, err := lab.CICancel(projectID, curJob.ID)
			if err != nil {
				a.Stop()
				log.Fatal(err)
			}
			curJob = job
			root.RemovePage("logs-" + curJob.Name)
			queueInputUpdate()
			a.ForceDraw()
		case 'p', 'r':
			if modalVisible {
				break
			}
			modalVisible = true
			modal := tview.NewModal().
				SetText(fmt.Sprintf("Are you sure you want to run %s", curJob.Name)).
				AddButtons([]string{"No", "Yes"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					modalVisible = false
					root.RemovePage("yesno")
					if buttonLabel == "No" {
						a.ForceDraw()
						return
					}
					root.RemovePage("logs-" + curJob.Name)
					a.ForceDraw()

					job, err := lab.CIPlayOrRetry(projectID, curJob.ID, curJob.Status)
					if err != nil {
						a.Stop()
						log.Fatal(err)
					}
					if job != nil {
						curJob = job
						a.ForceDraw()
					}
				})
			root.AddAndSwitchToPage("yesno", modal, false)
			queueInputUpdate()
			a.ForceDraw()
			return nil
		case 't':
			logsVisible = !logsVisible
			if !logsVisible {
				root.HidePage("logs-" + curJob.Name)
			}
			queueInputUpdate()
			a.ForceDraw()
			return nil
		case 'T':
			a.Suspend(func() {
				ctx, cancel := context.WithCancel(context.Background())
				go func() {
					err := doTrace(ctx, os.Stdout, projectID, pipelineID, curJob.Name)
					if err != nil {
						a.Stop()
						log.Fatal(err)
					}
					if ctx.Err() == nil { // not done or cancelled
						fmt.Println("\nPush <Enter> to resume ci view")
					}
				}()
				reader := bufio.NewReader(os.Stdin)
				for {
					r, _, err := reader.ReadRune()
					if err != io.EOF && err != nil {
						a.Stop()
						log.Fatal(err)
					}
					if r == '\n' {
						cancel()
						break
					}
				}
			})
			queueInputUpdate()
			return nil
		}
		queueInputUpdate()
		return event
	}
}

var (
	logsVisible, modalVisible bool
	curJob                    *gitlab.Job
	jobs                      []*gitlab.Job
	boxes                     map[string]*tview.TextView
)

// navigator manages the internal state for processing tcell.EventKeys
type navigator struct {
	depth, idx int
}

// Navigate uses the ci stages as boundaries and returns the currently focused
// job index after processing a *tcell.EventKey
func (n *navigator) Navigate(jobs []*gitlab.Job, event *tcell.EventKey) *gitlab.Job {
	stage := jobs[n.idx].Stage
	prev, next := adjacentStages(jobs, stage)
	switch event.Key() {
	case tcell.KeyLeft:
		stage = prev
	case tcell.KeyRight:
		stage = next
	}
	switch event.Rune() {
	case 'h':
		stage = prev
	case 'l':
		stage = next
	}
	l, u := stageBounds(jobs, stage)

	switch event.Key() {
	case tcell.KeyDown:
		n.depth++
		if n.depth > u-l {
			n.depth = u - l
		}
	case tcell.KeyUp:
		n.depth--
	}
	switch event.Rune() {
	case 'j':
		n.depth++
		if n.depth > u-l {
			n.depth = u - l
		}
	case 'k':
		n.depth--
	case 'g':
		n.depth = 0
	case 'G':
		n.depth = u - l
	}

	if n.depth < 0 {
		n.depth = 0
	}
	n.idx = l + n.depth
	if n.idx > u {
		n.idx = u
	}
	return jobs[n.idx]
}

func stageBounds(jobs []*gitlab.Job, s string) (l, u int) {
	if len(jobs) <= 1 {
		return 0, 0
	}
	p := jobs[0].Stage
	for i, v := range jobs {
		if v.Stage != s && u != 0 {
			return
		}
		if v.Stage != p {
			l = i
			p = v.Stage
		}
		if v.Stage == s {
			u = i
		}
	}
	return
}

func adjacentStages(jobs []*gitlab.Job, s string) (p, n string) {
	if len(jobs) == 0 {
		return "", ""
	}
	p = jobs[0].Stage

	for _, v := range jobs {
		if v.Stage != s && n != "" {
			n = v.Stage
			return
		}
		if v.Stage == s {
			n = "cur"
		}
		if n == "" {
			p = v.Stage
		}
	}
	n = jobs[len(jobs)-1].Stage
	return
}

func jobsView(app *tview.Application, jobsCh chan []*gitlab.Job, inputCh chan struct{}, root *tview.Pages) {
	select {
	case jobs = <-jobsCh:
	case <-inputCh:
	case <-time.NewTicker(time.Second * 1).C:
	}
	if jobs == nil {
		jobs = <-jobsCh
	}
	if curJob == nil && len(jobs) > 0 {
		curJob = jobs[0]
	}
	if modalVisible {
		return
	}
	if logsVisible {
		logsKey := "logs-" + curJob.Name
		if !root.SwitchToPage(logsKey).HasPage(logsKey) {
			tv := tview.NewTextView()
			tv.SetDynamicColors(true)
			tv.SetBorderPadding(0, 0, 1, 1).SetBorder(true)

			go func() {
				err := doTrace(context.Background(), vtclean.NewWriter(tview.ANSIWriter(tv), true), projectID, pipelineID, curJob.Name)
				if err != nil {
					app.Stop()
					log.Fatal(err)
				}
			}()
			root.AddAndSwitchToPage("logs-"+curJob.Name, tv, true)
		}
		return
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
		if tview.TaggedStringWidth(title) > maxTitle {
			b.SetTitleAlign(tview.AlignLeft)
		}
		if j.StartedAt != nil {
			end := time.Now()
			if j.FinishedAt != nil {
				end = *j.FinishedAt
			}
			b.SetText("\n" + fmtDuration(end.Sub(*j.StartedAt)))
			b.SetTextAlign(tview.AlignRight)
		} else {
			b.SetText("")
		}
		rowIdx++

	}
	root.SendToFront("jobs-" + curJob.Name)

}
func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02dm %02ds", m, s)
}
func box(root *tview.Pages, key string, x, y, w, h int) *tview.TextView {
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

func updateJobs(app *tview.Application, jobsCh chan []*gitlab.Job) {
	defer recoverPanic(app)
	for {
		if modalVisible {
			time.Sleep(time.Second * 1)
			continue
		}
		jobStructList, err := lab.CIJobs(projectID, pipelineID, followBridge)
		if len(jobStructList) == 0 || err != nil {
			app.Stop()
			log.Fatal(errors.Wrap(err, "failed to find ci jobs"))
		}

		jobs := make([]*gitlab.Job, 0)
		for _, jobStruct := range jobStructList {
			jobs = append(jobs, jobStruct.Job)
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
	if logsVisible || modalVisible {
		return nil
	}
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
	ciViewCmd.Flags().Bool("merge-request", false, "use merge request pipeline if enabled")
	ciCmd.AddCommand(ciViewCmd)
	carapace.Gen(ciViewCmd).PositionalCompletion(
		action.Remotes(),
		action.RemoteBranches(0),
	)
}
