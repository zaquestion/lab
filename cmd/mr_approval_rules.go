package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"

	gitlab "github.com/xanzy/go-gitlab"
)

func mrApprovalRuleShow(rule *gitlab.MergeRequestApprovalRule) {
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
				users += ","
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
				users += ","
			}
		}
		fmt.Println("   Approved By:", users)
	} else {
		fmt.Println("   Approved By:")
	}
}

var mrApprovalRuleCmd = &cobra.Command{
	Use:     "approval-rule [remote] [<MR id or branch>]",
	Aliases: []string{},
	Example: heredoc.Doc(`
		lab mr approval-rule 1234
		lab mr approval-rule 1234 --name "Fancy rule name"
		lab mr approval-rule --create --name "Fancy rule name" --user "prarit" --user "zaquestion"
		lab mr approval-rule --create --name "Fancy rule name" --user "prarit" --user "zaquestion" --approvals-required 1
		lab mr approval-rule --delete "Fancy rule name"`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsWithGitBranchMR(args)
		if err != nil {
			log.Fatal(err)
		}

		create, err := cmd.Flags().GetBool("create")
		if err != nil {
			log.Fatal(err)
		}

		if create {
			_approvalsRequired, err := cmd.Flags().GetString("approvals-required")
			if err != nil {
				log.Fatal(err)
			}
			approvalsRequired, err := strconv.Atoi(_approvalsRequired)
			if err != nil {
				log.Fatal(err)
			}

			name, err := cmd.Flags().GetString("name")
			if err != nil {
				log.Fatal(err)
			}
			if name == "" {
				fmt.Println("The --name option must be used with the --create option")
				os.Exit(1)
			}

			users, err := cmd.Flags().GetStringSlice("user")
			if err != nil {
				log.Fatal(err)
			}
			userIDs := getUserIDs(users)

			groups, err := cmd.Flags().GetStringSlice("group")
			if err != nil {
				log.Fatal(err)
			}
			groupIDs := getUserIDs(groups)

			rule, err := lab.CreateMRApprovalRule(rn, approvalsRequired, int(id), name, 0, groupIDs, userIDs)
			if err != nil {
				log.Fatal(err)
			}

			mrApprovalRuleShow(rule)
			return
		}

		deleteRule, err := cmd.Flags().GetString("delete")
		if err != nil {
			log.Fatal(err)
		}
		if deleteRule != "" {
			msg, err := lab.DeleteMRApprovalRule(rn, deleteRule, int(id))
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(msg)
			return
		}

		// default, no options just show the rules
		approvalState, err := lab.GetMRApprovalState(rn, int(id))
		if err != nil {
			log.Fatal(err)
		}

		name, err := cmd.Flags().GetString("name")
		if err != nil {
			log.Fatal(err)
		}

		for _, rule := range approvalState.Rules {
			if name != "" {
				if rule.Name != name {
					continue
				}
			}
			mrApprovalRuleShow(rule)
		}
	},
}

func init() {
	mrApprovalRuleCmd.Flags().BoolP("create", "c", false, "create a new rule.  See 'create:' sub-options in help")
	mrApprovalRuleCmd.Flags().StringP("delete", "d", "", "delete the named rule")
	mrApprovalRuleCmd.Flags().String("approvals-required", "0", "create: number of approvals required")
	mrApprovalRuleCmd.Flags().StringP("name", "n", "", "create: rule name (can also be used to display a rule)")
	mrApprovalRuleCmd.Flags().String("project-rule-id", "", "create: project rule id for new rule")
	mrApprovalRuleCmd.Flags().StringSliceP("user", "u", []string{}, "create: approvers for new rule; can be used multiple times for different users")
	mrApprovalRuleCmd.Flags().StringSliceP("group", "g", []string{}, "create: groups for new rule; can be used multiple times for different groups")

	mrCmd.AddCommand(mrApprovalRuleCmd)
}
