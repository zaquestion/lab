package cmd

import (
	"strconv"
	"testing"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
)

func assertScreen(t *testing.T, screen tcell.Screen, expected []string) {
	sx, sy := screen.Size()
	assert.Equal(t, len(expected), sy)
	assert.Equal(t, len([]rune(expected[0])), sx)
	actual := make([]string, sy)
	for y, str := range expected {
		runes := make([]rune, len(str))
		row := []rune(str)
		for x, expectedRune := range row {
			r, _, _, _ := screen.GetContent(x, y)
			runes[x] = r
			assert.Equal(t, expectedRune, r, "%s != %s at (%d,%d)",
				strconv.QuoteRune(expectedRune), strconv.QuoteRune(r), x, y)
		}
		actual[y] = string(runes)
	}
	t.Logf("Expected w: %d l: %d", len([]rune(expected[0])), len(expected))
	for _, str := range expected {
		t.Log(str)
	}
	t.Logf("Actual w: %d l: %d", len([]rune(actual[0])), len(actual))
	for _, str := range actual {
		t.Log(str)
	}
}

func Test_line(t *testing.T) {
	tests := []struct {
		desc     string
		lineF    func(screen tcell.Screen, x, y, l int)
		x, y, l  int
		expected []string
	}{
		{
			"hline",
			hline,
			2, 2, 5,
			[]string{
				"          ",
				"          ",
				"  ━━━━━   ",
				"          ",
				"          ",
				"          ",
				"          ",
				"          ",
				"          ",
				"          ",
			},
		},
		{
			"hline overflow",
			hline,
			2, 2, 10,
			[]string{
				"          ",
				"          ",
				"  ━━━━━━━━",
				"          ",
				"          ",
				"          ",
				"          ",
				"          ",
				"          ",
				"          ",
			},
		},
		{
			"vline",
			vline,
			2, 2, 5,
			[]string{
				"          ",
				"          ",
				"  ┃       ",
				"  ┃       ",
				"  ┃       ",
				"  ┃       ",
				"  ┃       ",
				"          ",
				"          ",
				"          ",
			},
		},
		{
			"vline overflow",
			vline,
			2, 2, 10,
			[]string{
				"          ",
				"          ",
				"  ┃       ",
				"  ┃       ",
				"  ┃       ",
				"  ┃       ",
				"  ┃       ",
				"  ┃       ",
				"  ┃       ",
				"  ┃       ",
			},
		},
	}

	for _, test := range tests {
		screen := tcell.NewSimulationScreen("UTF-8")
		err := screen.Init()
		if err != nil {
			t.Fatal(err)
		}
		// Set screen to matrix size
		screen.SetSize(len(test.expected), len(test.expected[0]))

		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			test.lineF(screen, test.x, test.y, test.l)
			screen.Show()
			assertScreen(t, screen, test.expected)
		})
	}
}

func testbox(x, y, w, h int) *tview.TextView {
	b := tview.NewTextView()
	b.SetBorder(true)
	b.SetRect(x, y, w, h)
	return b
}

func Test_connect(t *testing.T) {
	tests := []struct {
		desc        string
		b1, b2      *tview.Box
		first, last bool
		expected    []string
	}{
		{
			"first stage",
			testbox(2, 1, 3, 3).Box, testbox(2, 5, 3, 3).Box,
			true, false,
			[]string{
				"          ",
				"  ┌─┐     ",
				"  │ │     ",
				"  └─┘ ┃   ",
				"      ┃   ",
				"  ┌─┐ ┃   ",
				"  │ │━┛   ",
				"  └─┘     ",
				"          ",
				"          ",
			},
		},
		{
			"last stage",
			testbox(5, 1, 3, 3).Box, testbox(5, 5, 3, 3).Box,
			false, true,
			[]string{
				"          ",
				"     ┌─┐  ",
				"   ┳ │ │  ",
				"   ┃ └─┘  ",
				"   ┃      ",
				"   ┃ ┌─┐  ",
				"   ┗━│ │  ",
				"     └─┘  ",
				"          ",
				"          ",
			},
		},
		{
			"cross stage",
			testbox(1, 1, 3, 3).Box, testbox(7, 1, 3, 3).Box,
			false, false,
			[]string{
				"          ",
				" ┌─┐   ┌─┐",
				" │ │━━━│ │",
				" └─┘   └─┘",
				"          ",
				"          ",
				"          ",
				"          ",
				"          ",
				"          ",
			},
		},
	}

	for _, test := range tests {
		screen := tcell.NewSimulationScreen("UTF-8")
		err := screen.Init()
		if err != nil {
			t.Fatal(err)
		}
		// Set screen to matrix size
		screen.SetSize(len(test.expected), len(test.expected[0]))

		test.b1.Draw(screen)
		test.b2.Draw(screen)

		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			connect(screen, test.b1, test.b2, 2, test.first, test.last)
			screen.Show()
			assertScreen(t, screen, test.expected)
		})
	}
}

func Test_connectJobs(t *testing.T) {
	expected := []string{
		"                 ",
		" ┌─┐   ┌─┐   ┌─┐ ",
		" │ │┳━┳│ │┳━┳│ │ ",
		" └─┘┃ ┃└─┘┃ ┃└─┘ ",
		"    ┃ ┃   ┃ ┃    ",
		" ┌─┐┃ ┃┌─┐┃ ┃┌─┐ ",
		" │ │┫ ┣│ │┫ ┗│ │ ",
		" └─┘┃ ┃└─┘┃  └─┘ ",
		"    ┃ ┃   ┃      ",
		" ┌─┐┃ ┃┌─┐┃      ",
		" │ │┫ ┗│ │┛      ",
		" └─┘┃  └─┘       ",
		"    ┃            ",
		" ┌─┐┃            ",
		" │ │┛            ",
		" └─┘             ",
		"                 ",
	}
	jobs := []*gitlab.Job{
		&gitlab.Job{
			Name:  "stage1-job1",
			Stage: "stage1",
		},
		&gitlab.Job{
			Name:  "stage1-job2",
			Stage: "stage1",
		},
		&gitlab.Job{
			Name:  "stage1-job3",
			Stage: "stage1",
		},
		&gitlab.Job{
			Name:  "stage1-job4",
			Stage: "stage1",
		},
		&gitlab.Job{
			Name:  "stage2-job1",
			Stage: "stage2",
		},
		&gitlab.Job{
			Name:  "stage2-job2",
			Stage: "stage2",
		},
		&gitlab.Job{
			Name:  "stage2-job3",
			Stage: "stage2",
		},
		&gitlab.Job{
			Name:  "stage3-job1",
			Stage: "stage3",
		},
		&gitlab.Job{
			Name:  "stage3-job2",
			Stage: "stage3",
		},
	}
	boxes := map[string]*tview.TextView{
		"jobs-stage1-job1": testbox(1, 1, 3, 3),
		"jobs-stage1-job2": testbox(1, 5, 3, 3),
		"jobs-stage1-job3": testbox(1, 9, 3, 3),
		"jobs-stage1-job4": testbox(1, 13, 3, 3),

		"jobs-stage2-job1": testbox(7, 1, 3, 3),
		"jobs-stage2-job2": testbox(7, 5, 3, 3),
		"jobs-stage2-job3": testbox(7, 9, 3, 3),

		"jobs-stage3-job1": testbox(13, 1, 3, 3),
		"jobs-stage3-job2": testbox(13, 5, 3, 3),
	}

	screen := tcell.NewSimulationScreen("UTF-8")
	err := screen.Init()
	if err != nil {
		t.Fatal(err)
	}
	// Set screen to matrix size
	screen.SetSize(len(expected), len(expected[0]))

	for _, b := range boxes {
		b.Draw(screen)
	}

	err = connectJobs(screen, jobs, boxes)
	if err != nil {
		t.Fatal(err)
	}

	screen.Show()
	assertScreen(t, screen, expected)
}

func Test_connectJobsNegative(t *testing.T) {
	tests := []struct {
		desc  string
		jobs  []*gitlab.Job
		boxes map[string]*tview.TextView
	}{
		{
			"determinePadding -- first job missing",
			[]*gitlab.Job{
				&gitlab.Job{
					Name:  "stage1-job1",
					Stage: "stage1",
				},
			},
			map[string]*tview.TextView{
				"jobs-stage2-job1": testbox(1, 5, 3, 3),
				"jobs-stage2-job2": testbox(1, 9, 3, 3),
			},
		},
		{
			"determinePadding -- second job missing",
			[]*gitlab.Job{
				&gitlab.Job{
					Name:  "stage1-job1",
					Stage: "stage1",
				},
				&gitlab.Job{
					Name:  "stage2-job1",
					Stage: "stage2",
				},
				&gitlab.Job{
					Name:  "stage2-job2",
					Stage: "stage2",
				},
			},
			map[string]*tview.TextView{
				"jobs-stage1-job1": testbox(1, 1, 3, 3),
				"jobs-stage2-job2": testbox(1, 9, 3, 3),
			},
		},
		{
			"connect -- third job missing",
			[]*gitlab.Job{
				&gitlab.Job{
					Name:  "stage1-job1",
					Stage: "stage1",
				},
				&gitlab.Job{
					Name:  "stage2-job1",
					Stage: "stage2",
				},
				&gitlab.Job{
					Name:  "stage2-job2",
					Stage: "stage2",
				},
			},
			map[string]*tview.TextView{
				"jobs-stage1-job1": testbox(1, 1, 3, 3),
				"jobs-stage2-job1": testbox(1, 5, 3, 3),
			},
		},
		{
			"connect -- third job missing",
			[]*gitlab.Job{
				&gitlab.Job{
					Name:  "stage1-job1",
					Stage: "stage1",
				},
				&gitlab.Job{
					Name:  "stage2-job1",
					Stage: "stage2",
				},
				&gitlab.Job{
					Name:  "stage2-job2",
					Stage: "stage2",
				},
			},
			map[string]*tview.TextView{
				"jobs-stage1-job1": testbox(1, 1, 3, 3),
				"jobs-stage2-job1": testbox(1, 5, 3, 3),
			},
		},
	}
	for _, test := range tests {
		screen := tcell.NewSimulationScreen("UTF-8")
		err := screen.Init()
		if err != nil {
			t.Fatal(err)
		}
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			assert.Error(t, connectJobs(screen, test.jobs, test.boxes))

		})
	}
}

func Test_jobsView(t *testing.T) {
	expected := []string{
		"  ┌────────────────────┐      ┌────────────────────┐      ┌────────────────────┐        ",
		"  │       Stage1       │      │       Stage2       │      │       Stage3       │        ",
		"  └────────────────────┘      └────────────────────┘      └────────────────────┘        ",
		"                                                                                        ",
		"  ┌✔ stage1-job1-reall…┐      ┌───● stage2-job1────┐      ┌───● stage3-job1────┐        ",
		"  │                    │      │                    │      │                    │        ",
		"  │             01m 01s│━┳━━┳━│                    │━┳━━┳━│                    │        ",
		"  └────────────────────┘ ┃  ┃ └────────────────────┘ ┃  ┃ └────────────────────┘        ",
		"                         ┃  ┃                        ┃  ┃                               ",
		"  ┌───✔ stage1-job2────┐ ┃  ┃ ┌───● stage2-job2────┐ ┃  ┃ ┌───● stage3-job2────┐        ",
		"  │                    │ ┃  ┃ │                    │ ┃  ┃ │                    │        ",
		"  │                    │━┫  ┣━│                    │━┫  ┗━│                    │        ",
		"  └────────────────────┘ ┃  ┃ └────────────────────┘ ┃    └────────────────────┘        ",
		"                         ┃  ┃                        ┃                                  ",
		"  ┌───✔ stage1-job3────┐ ┃  ┃ ┌───● stage2-job3────┐ ┃                                  ",
		"  │                    │ ┃  ┃ │                    │ ┃                                  ",
		"  │                    │━┫  ┗━│                    │━┛                                  ",
		"  └────────────────────┘ ┃    └────────────────────┘                                    ",
		"                         ┃                                                              ",
		"  ┌───✘ stage1-job4────┐ ┃                                                              ",
		"  │                    │ ┃                                                              ",
		"  │                    │━┛                                                              ",
		"  └────────────────────┘                                                                ",
		"                                                                                        ",
		"                                                                                        ",
		"                                                                                        ",
	}
	now := time.Now()
	past := now.Add(time.Second * -61)
	jobs := []*gitlab.Job{
		&gitlab.Job{
			Name:       "stage1-job1-really-long",
			Stage:      "stage1",
			Status:     "success",
			StartedAt:  &past, // relies on test running in <1s we'll see how it goes
			FinishedAt: &now,
		},
		&gitlab.Job{
			Name:   "stage1-job2",
			Stage:  "stage1",
			Status: "success",
		},
		&gitlab.Job{
			Name:   "stage1-job3",
			Stage:  "stage1",
			Status: "success",
		},
		&gitlab.Job{
			Name:   "stage1-job4",
			Stage:  "stage1",
			Status: "failed",
		},
		&gitlab.Job{
			Name:   "stage2-job1",
			Stage:  "stage2",
			Status: "running",
		},
		&gitlab.Job{
			Name:   "stage2-job2",
			Stage:  "stage2",
			Status: "running",
		},
		&gitlab.Job{
			Name:   "stage2-job3",
			Stage:  "stage2",
			Status: "pending",
		},
		&gitlab.Job{
			Name:   "stage3-job1",
			Stage:  "stage3",
			Status: "manual",
		},
		&gitlab.Job{
			Name:   "stage3-job2",
			Stage:  "stage3",
			Status: "manual",
		},
	}

	boxes = make(map[string]*tview.TextView)
	jobsCh := make(chan []*gitlab.Job)
	root := tview.NewPages()
	root.SetBorderPadding(1, 1, 2, 2)

	screen := tcell.NewSimulationScreen("UTF-8")
	err := screen.Init()
	if err != nil {
		t.Fatal(err)
	}
	// Set screen to matrix size
	screen.SetSize(len([]rune(expected[0])), len(expected))
	w, h := screen.Size()
	root.SetRect(0, 0, w, h)

	go func() {
		jobsCh <- jobs
	}()
	jobsView(nil, jobsCh, root)(screen)
	root.Draw(screen)
	connectJobsView(nil)(screen)
	screen.Sync()
	assertScreen(t, screen, expected)
}

func Test_latestJobs(t *testing.T) {
	tests := []struct {
		desc     string
		jobs     []*gitlab.Job
		expected []*gitlab.Job
	}{
		{
			desc: "no newer jobs",
			jobs: []*gitlab.Job{
				&gitlab.Job{
					ID:    1,
					Name:  "stage1-job1",
					Stage: "stage1",
				},
				&gitlab.Job{
					ID:    2,
					Name:  "stage1-job2",
					Stage: "stage1",
				},
				&gitlab.Job{
					ID:    3,
					Name:  "stage1-job3",
					Stage: "stage1",
				},
			},
			expected: []*gitlab.Job{
				&gitlab.Job{
					ID:    1,
					Name:  "stage1-job1",
					Stage: "stage1",
				},
				&gitlab.Job{
					ID:    2,
					Name:  "stage1-job2",
					Stage: "stage1",
				},
				&gitlab.Job{
					ID:    3,
					Name:  "stage1-job3",
					Stage: "stage1",
				},
			},
		},
		{
			desc: "1 newer",
			jobs: []*gitlab.Job{
				&gitlab.Job{
					ID:    1,
					Name:  "stage1-job1",
					Stage: "stage1",
				},
				&gitlab.Job{
					ID:    2,
					Name:  "stage1-job2",
					Stage: "stage1",
				},
				&gitlab.Job{
					ID:    3,
					Name:  "stage1-job3",
					Stage: "stage1",
				},
				&gitlab.Job{
					ID:    4,
					Name:  "stage1-job1",
					Stage: "stage1",
				},
			},
			expected: []*gitlab.Job{
				&gitlab.Job{
					ID:    4,
					Name:  "stage1-job1",
					Stage: "stage1",
				},
				&gitlab.Job{
					ID:    2,
					Name:  "stage1-job2",
					Stage: "stage1",
				},
				&gitlab.Job{
					ID:    3,
					Name:  "stage1-job3",
					Stage: "stage1",
				},
			},
		},
		{
			desc: "2 newer",
			jobs: []*gitlab.Job{
				&gitlab.Job{
					ID:    1,
					Name:  "stage1-job1",
					Stage: "stage1",
				},
				&gitlab.Job{
					ID:    2,
					Name:  "stage1-job2",
					Stage: "stage1",
				},
				&gitlab.Job{
					ID:    3,
					Name:  "stage1-job3",
					Stage: "stage1",
				},
				&gitlab.Job{
					ID:    4,
					Name:  "stage1-job3",
					Stage: "stage1",
				},
				&gitlab.Job{
					ID:    5,
					Name:  "stage1-job1",
					Stage: "stage1",
				},
			},
			expected: []*gitlab.Job{
				&gitlab.Job{
					ID:    5,
					Name:  "stage1-job1",
					Stage: "stage1",
				},
				&gitlab.Job{
					ID:    2,
					Name:  "stage1-job2",
					Stage: "stage1",
				},
				&gitlab.Job{
					ID:    4,
					Name:  "stage1-job3",
					Stage: "stage1",
				},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			jobs := latestJobs(test.jobs)
			assert.Equal(t, test.expected, jobs)
		})
	}
}
