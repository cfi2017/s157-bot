/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"github.com/bwmarrin/discordgo"
	"github.com/cfi2017/s157-bot/cmd/alliance"
	"github.com/cfi2017/s157-bot/cmd/role"
	"github.com/spf13/cobra"
)

var cfgFile string

func RootCmdFactory(s *discordgo.Session, e *discordgo.MessageCreate) *cobra.Command {
	cmd := &cobra.Command{
		Short: "Management bot for alliances in the stfc network.",
		Long: `Management bot for alliances in the stfc network.

!help for help.
`,
		// Uncomment the following line if your bare application
		// has an action associated with it:
		//	Run: func(cmd *cobra.Command, args []string) { },
	}
	cmd.AddCommand(pingCmdFactory(s, e))
	cmd.AddCommand(echoCmdFactory(s, e))
	cmd.AddCommand(alliance.RootCmdFactory(s, e))
	cmd.AddCommand(role.RootCmdFactory(s, e))
	return cmd
}
