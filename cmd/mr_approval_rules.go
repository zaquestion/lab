package cmd

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"

	gitlab "github.com/xanzy/go-gitlab"
)

func mrApprovalRuleShow(approvalState *gitlab.MergeRequestApprovalState) {
	for _, rule := range approvalState.Rules {
		fmt.Println("Rule:", rule.Name)
		if rule.Approved {
			fmt.Println("   Approved: Y")
		} else {
			fmt.Println("   Approved:")
		}

		if rule.RuleType == "regular" {
			users := ""
			for u, user := range rule.EligibleApprovers {
				users += fmt.Sprintf("%s", user.Username)
				if u != (len(rule.EligibleApprovers) - 1) {
					users +=","
				}
			}
			fmt.Println("   Approvers:", users)
		} else if rule.RuleType == "any_approver" {
			fmt.Println("   Approvers: All eligible users")
		}

		if rule.ApprovalsRequired > 0 {
			fmt.Printf("   Approvals: %d of %d\n", len(rule.ApprovedBy), rule.ApprovalsRequired)
		} else {
			fmt.Println("   Approvals: Optional")
		}

		if len(rule.ApprovedBy) > 0 {
			users := ""
			for u, user := range rule.ApprovedBy {
				users += fmt.Sprintf("%s", user.Username)
				if u != (len(rule.ApprovedBy) - 1) {
					users +=","
				}
			}
			fmt.Println("   Approved By:", users)
		} else {
			fmt.Println("   Approved By:")
		}
	}
}

var mrApprovalRuleCmd = &cobra.Command{
	Use:     "approval-rule [remote] [<MR id or branch>]",
	Aliases: []string{},
	Example: heredoc.Doc(`
		lab mr approval-rule 1234`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsWithGitBranchMR(args)
		if err != nil {
			log.Fatal(err)
		}

		approvalState, err := lab.GetMRApprovalState(rn, int(id))
		if err != nil {
			log.Fatal(err)
		}

		// default, no options just show the rules
		mrApprovalRuleShow(approvalState)
	},
}

func init() {
	mrCmd.AddCommand(mrApprovalRuleCmd)
}
