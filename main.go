package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Necroforger/dgwidgets"
	"github.com/bwmarrin/discordgo"
)

const (
	MEMBER_NAME = "Alliance"
	LEADER_NAME = "Representative"
)

var (
	token   = flag.String("token", "", "bot token")
	fPrefix = flag.String("p", "!", "bot prefix")
	session *discordgo.Session
)

/*
verification flow:

*/
func main() {
	flag.Parse()

	var err error
	session, err = discordgo.New("Bot " + *token)
	if err != nil {
		panic(err)
	}

	// add handlers
	session.AddHandlerOnce(onReady)
	session.AddHandler(onMessageCreate)
	session.AddHandler(onGuildJoin)
	session.AddHandler(onGuildLeave)

	// open websocket
	err = session.Open()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("STFC is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	session.Close()

}

func onReady(s *discordgo.Session, event *discordgo.Ready) {

	// Set the playing status.
	s.UpdateStatus(0, "!startrek")
}

func onMessageCreate(s *discordgo.Session, event *discordgo.MessageCreate) {
	if event.WebhookID != "" {
		return
	}
	if event.Author.Bot {
		return
	}

	// is command?
	if strings.HasPrefix(strings.TrimSpace(event.Content), *fPrefix) {
		commands := strings.Split(strings.TrimPrefix(strings.TrimSpace(event.Content), *fPrefix), " ")
		if len(commands) == 0 {
			sendMessage(event.ChannelID, "invalid command.")
			return
		}

		switch commands[0] {
		case "guild":
			handleGuildCommand(event, commands[1:])
			break
		default:
			// sendMessage(event.ChannelID, "invalid command.")
			return
		}

	}

}

func onGuildJoin(s *discordgo.Session, event *discordgo.GuildMemberAdd) {
	if s.State.User.ID == event.User.ID {
		// self join
		g, err := s.Guild(event.GuildID)
		if err != nil {
			log.Println(err)
			return
		}
		ch, err := s.UserChannelCreate(g.OwnerID)
		if err != nil {
			log.Println(err)
			return
		}
		_, err = s.ChannelMessageSend(ch.ID, "Hello. To set up, please make sure the bot has access to a private, empty text channel called #role-bot.")
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func onGuildLeave(s *discordgo.Session, event *discordgo.GuildMemberRemove) {
	if event.Roles != nil {
		if ok, _ := HasRole(event.Member, LEADER_NAME); ok {
			// todo: send message to admins
		}
	}
}

type PendingRequest struct {
	ChannelID   string // the channel id of the leader being prompted
	MessageID   string // the message containing the prompt
	LeaderID    string // the leader being prompted
	RequesterID string // the requester prompting someone to accept them into the guild
}

func sendMessage(channelId string, message string) {
	_, err := session.ChannelMessageSend(channelId, message)
	if err != nil {
		log.Println(err)
	}
	return
}

func handleGuildCommand(event *discordgo.MessageCreate, args []string) {

	if len(args) < 1 {
		sendMessage(event.ChannelID, `invalid command. example usages: `+"```"+`
!guild <tag> <username> -- join a guild
!guild leave -- leave a guild
!guild promote @username -- make another person representative`+"```")
		return
	}

	if IsDM(event.ChannelID) {
		sendMessage(event.ChannelID, "Please run this command on a server.")
		return
	}

	switch args[0] {
	case "leave":
		leaveGuild(event)
		break
	case "promote":
		if len(args) != 2 {
			sendMessage(event.ChannelID, "Please specify your successor.")
			return
		}
		promote(event, args[1])
		break
	default:
		if len(args) == 0 {
			sendMessage(event.ChannelID, "invalid command. usage: !guild <tag> <username>")
			return
		}
		if len(args[0]) > 4 {
			sendMessage(event.ChannelID, "your alliance tag must be at most four characters")
			return
		}
		if len(args) == 1 {
			joinGuild(event, args[0], event.Author.Username)
			return
		}
		if len(args) != 2 {
			sendMessage(event.ChannelID, "invalid command. usage: !guild <tag> <username>")
			return
		}
		joinGuild(event, args[0], args[1])
	}
}

func promote(event *discordgo.MessageCreate, s string) {
	// is rep?
	event.Member.GuildID = event.GuildID
	guildLeader, leaderID := HasRole(event.Member, LEADER_NAME)
	if !guildLeader {
		sendMessage(event.ChannelID, "You are not the leader of your guild.")
		return
	}

	if len(event.Mentions) != 1 {
		sendMessage(event.ChannelID, "Please mention exactly one successor.")
		return
	}

	target := event.Mentions[0]
	targetMember, _ := session.GuildMember(event.GuildID, target.ID)
	targetMember.GuildID = event.GuildID
	ok, _ := HasRole(targetMember, MEMBER_NAME)
	if !ok {
		sendMessage(event.ChannelID, "Your target is not in a guild.")
		return
	}

	if !strings.HasPrefix(targetMember.Nick, strings.Split(event.Member.Nick, " ")[0]) {
		sendMessage(event.ChannelID, "Your target is not in the same guild as you.")
		return
	}

	session.GuildMemberRoleAdd(event.GuildID, event.Mentions[0].ID, leaderID)
	session.GuildMemberRoleRemove(event.GuildID, event.Author.ID, leaderID)
	sendMessage(event.ChannelID, "Done.")

}

// if the message is a private message, error
// if a user is not in a guild, error
// if a user is a representative, error and have them choose a different rep
// if a user is the only guild member, remove any tags
// else remove any tags
func leaveGuild(event *discordgo.MessageCreate) {
	channel, err := session.Channel(event.ChannelID)
	if err != nil {
		log.Println(err)
		return
	}
	if channel.Type == discordgo.ChannelTypeDM {
		_, err = session.ChannelMessageSend(event.ChannelID, "Error: This command can only be run on a server.")
		if err != nil {
			log.Println(err)
		}
		return
	}
	gMember, err := session.GuildMember(event.GuildID, event.Author.ID)
	if err != nil {
		log.Println(err)
		return
	}
	gMember.GuildID = event.GuildID
	guildLeader, leaderID := HasRole(gMember, LEADER_NAME)
	guildMember, memberID := HasRole(gMember, MEMBER_NAME)
	if !guildMember {
		_, err = session.ChannelMessageSend(channel.ID, "You are not a member of any guild.")
		return
	}
	if guildLeader {
		// todo: check for other guild members
		members := GetMembers(event.GuildID)
		m := make([]*discordgo.Member, 0)
		for _, member := range members {
			for _, r := range member.Roles {
				if r == memberID && strings.HasPrefix(member.Nick, strings.Split(event.Member.Nick, " ")[0]) && member.User.ID != event.Author.ID {
					m = append(m, member)
				}
			}
		}
		if len(m) > 0 {
			session.ChannelMessageSend(channel.ID, "Your current alliance has members. Promote someone before leaving.")
			return
		}
		session.GuildMemberRoleRemove(event.GuildID, event.Author.ID, leaderID)
	}
	session.GuildMemberRoleRemove(event.GuildID, event.Author.ID, memberID)
	err = session.GuildMemberNickname(event.GuildID, event.Author.ID, strings.Split(event.Member.Nick, "] ")[1])
	if err != nil {
		sendMessage(event.ChannelID, "couldn't change your nickname.")
	}
}

func joinGuild(event *discordgo.MessageCreate, tag, user string) {
	tag = strings.ToUpper(tag)
	if IsDM(event.ChannelID) {
		session.ChannelMessageSend(event.ChannelID, "Please use this command on a server.")
		return
	}
	event.Member.GuildID = event.GuildID
	hasMemberRole, _ := HasRole(event.Member, MEMBER_NAME)
	if hasMemberRole {
		session.ChannelMessageSend(event.ChannelID, "You are already in an alliance.")
		return
	}

	// check for members in same guild
	guildMembers := GetMembers(event.GuildID)
	leaderID := GetRoleID(event.GuildID, LEADER_NAME)
	memberID := GetRoleID(event.GuildID, MEMBER_NAME)
	var leader *discordgo.Member
	// if none - add and make rep
	for _, member := range guildMembers {
		for _, role := range member.Roles {
			if role == leaderID {
				if strings.HasPrefix(member.Nick, "["+tag+"]") {
					leader = member
				}
			}
		}
	}
	if leader == nil {
		// add and make rep
		err := session.GuildMemberNickname(event.GuildID, event.Author.ID, "["+tag+"] "+user)
		if err != nil {
			log.Println(err)
		}
		err = session.GuildMemberRoleAdd(event.GuildID, event.Author.ID, memberID)
		if err != nil {
			sendMessage(event.ChannelID, "couldn't change your nickname.")
		}
		err = session.GuildMemberRoleAdd(event.GuildID, event.Author.ID, leaderID)
		if err != nil {
			log.Println(err)
		}
	} else {
		session.ChannelMessageSend(event.ChannelID, "asking your representative for permission")
		channel, err := session.UserChannelCreate(leader.User.ID)
		if err != nil {
			session.ChannelMessageSend(event.ChannelID, "couldn't ask your rep. please contact an admin.")
			return
		}
		e := &discordgo.MessageEmbed{
			Description: "Someone wants to join your alliance!",
		}

		w := dgwidgets.NewWidget(session, channel.ID, e)
		w.Handle("✅", func(widget *dgwidgets.Widget, reaction *discordgo.MessageReaction) {
			// adding new user to guild
			session.GuildMemberNickname(event.GuildID, event.Author.ID, "["+tag+"] "+user)
			session.GuildMemberRoleAdd(event.GuildID, event.Author.ID, memberID)
			// todo: confirm reps
		})
		w.Handle("", func(widget *dgwidgets.Widget, reaction *discordgo.MessageReaction) {
			session.ChannelMessageDelete(reaction.ChannelID, reaction.MessageID)
		})
		w.Spawn()

	}

}

func GetMembers(guild string) []*discordgo.Member {
	members := make([]*discordgo.Member, 0)
	m, err := session.GuildMembers(guild, "", 1000)
	members = append(members, m...)
	if err != nil {
		return members
	}
	for len(m) > 0 {
		m, err = session.GuildMembers(guild, m[len(m)-1].User.ID, 1000)
		if err != nil {
			return members
		}
		members = append(members, m...)
	}
	return members
}

func IsDM(channel string) bool {
	c, err := session.Channel(channel)
	if err != nil {
		return false
	}
	return c.Type == discordgo.ChannelTypeDM
}

func GetRoleID(guild, name string) string {
	roles, err := session.GuildRoles(guild)
	if err != nil {
		return ""
	}
	for _, role := range roles {
		if role.Name == name {
			return role.ID
		}
	}
	return ""
}

func HasRole(member *discordgo.Member, role string) (ok bool, rid string) {
	g, err := session.Guild(member.GuildID)
	if err != nil {
		return false, ""
	}
	for _, r := range g.Roles {
		if r.Name == role {
			rid = r.ID
		}
	}
	if rid == "" {
		return
	}
	for _, r := range member.Roles {
		if r == rid {
			return true, r
		}
	}
	return false, rid
}
