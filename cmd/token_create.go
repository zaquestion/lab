package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	expiresAt time.Time
)

var tokenCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "create a new Personal Access Token",
	Args:  cobra.MaximumNArgs(1),
	Example: heredoc.Doc(`
		lab token create`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		// The values of name and scopes must be specified, they are not optional.
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			log.Fatal("The name of the token must be specified.")
		}

		scopes, _ := cmd.Flags().GetStringSlice("scopes")
		if len(scopes) == 0 {
			log.Fatal("Scopes must be specified.  See --help for available options.")
		}

		// expiresat is optional
		expiresat, _ := cmd.Flags().GetString("expiresat")
		if expiresat != "" {
			s := strings.Split(expiresat, "-")
			if len(s) != 3 {
				log.Fatal("Incorrect date specified, must be YYYY-MM-DD format")
			}

			year, err := strconv.Atoi(s[0])
			if err != nil {
				log.Fatal("Invalid year specified")
			}
			month, err := strconv.Atoi(s[1])
			if err != nil {
				log.Fatal("Invalid month specified")
			}
			day, err := strconv.Atoi(s[2])
			if err != nil {
				log.Fatal("Invalid day specified")
			}

			loc, _ := time.LoadLocation("UTC")
			expiresAt = time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc)

			yearNow, monthNow, dayNow := time.Now().UTC().Date()
			yearFromNow := time.Date(yearNow, monthNow, dayNow, 0, 0, 0, 0, loc).AddDate(1, 0, 0)
			if expiresAt.After(yearFromNow) {
				log.Fatalf("Expires date can only be a maximum of one year from now (%s)", yearFromNow.String())
			}
		} else {
			loc, _ := time.LoadLocation("UTC")
			yearNow, monthNow, dayNow := time.Now().UTC().Date()
			expiresAt = time.Date(yearNow, monthNow, dayNow, 0, 0, 0, 0, loc).AddDate(0, 0, 30)
		}

		pat, err := lab.CreatePAT(name, expiresAt, scopes)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s created set to expire on %s", pat, expiresat)
	},
}

func init() {
	tokenCreateCmd.Flags().StringP("name", "n", "", "name of token")
	tokenCreateCmd.Flags().StringSliceP("scopes", "s", []string{}, "Comma separated scopes for this token. (Available scopes are: api, read_api, read_user, read_repository, write_repository, read_registry, write_registry, sudo, admin_mode.")
	tokenCreateCmd.Flags().StringP("expiresat", "e", "", "YYYY-MM-DD formatted date the token will expire on (Default: 30 days from now, Maximum: One year from today.)")
	tokenCmd.AddCommand(tokenCreateCmd)
}
