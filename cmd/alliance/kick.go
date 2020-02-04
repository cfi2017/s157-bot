package alliance

import (
	"github.com/bwmarrin/discordgo"
	"github.com/spf13/cobra"
)

func kickCmdFactory(s *discordgo.Session, e *discordgo.MessageCreate) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kick",
		Short: "Kicks a user from your alliance.",
		Long: `
Kicks a user out of your alliance.
`,
		Run: func(cmd *cobra.Command, args []string) {
			_, _ = s.ChannelMessageSend(e.ChannelID, "Kick called. "+s.LastHeartbeatAck.Sub(s.LastHeartbeatSent).String())
		},
	}
	return cmd
}
