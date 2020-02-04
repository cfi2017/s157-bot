package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/cobra"
)

func echoCmdFactory(_ *discordgo.Session, _ *discordgo.MessageCreate) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "echo",
		Short: "A brief description of your command",
		Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	}
	caps := cmd.Flags().BoolP("caps", "c", false, "full caps message")
	blacklist := cmd.Flags().StringSliceP("blacklist", "b", []string{}, "blacklist words")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		args = filterArray(args, *blacklist)
		msg := strings.Join(args, " ")
		if *caps {
			msg = strings.ToUpper(msg)
		}
		_, err := fmt.Fprintln(cmd.OutOrStdout(), msg)
		if err != nil {
			log.Println(err)
		}
	}
	return cmd
}

func filterArray(args []string, blacklist []string) []string {
	msgArgs := make([]string, 0)
	for _, arg := range args {
		blacklisted := false
		for _, s := range blacklist {
			if arg == s {
				blacklisted = true
				break
			}
		}
		if !blacklisted {
			msgArgs = append(msgArgs, arg)
		}
	}
	return msgArgs
}
