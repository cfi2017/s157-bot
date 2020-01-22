package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
	"github.com/cfi2017/dgwidgets"
)

const (
	BotOwnerId              = "135468266695950336"
	MemberRoleId            = "667160643014492182"
	LeaderRoleId            = "666371711616417836"
	CommodoreRoleId         = "666737950985420802"
	NameNotChanged          = "666986823662174228"
	NewsChannelId           = "667833607615676457" // "667300349110911008"
	ServerNewsQueueId       = "669565978044268554"
	GameNewsQueueId         = "669566515322028045"
	GameNewsRoleId          = "669566606472511488"
	ServerNewsRoleId        = "669560894774181908"
	ChannelAdmin            = "666274224565911571"
	CrawAssignmentChannelId = "667716602610974740"
)

var (
	token     = flag.String("token", "", "bot token")
	fPrefix   = flag.String("p", "!", "bot prefix")
	session   *discordgo.Session
	ErrNoData = errors.New("no such error")
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
	session.AddHandler(onGuildLeave)

	state, err := GetGuildState("665976752992157706")
	if err != nil {
		panic(err)
	}
	for _, prompt := range state.RolePrompts {
		w := &dgwidgets.Widget{
			Keys: []string{
				prompt.Emote,
			},
			Handlers: map[string]dgwidgets.WidgetHandler{
				prompt.Emote: func(widget *dgwidgets.Widget, reaction *discordgo.MessageReaction) {
					s := widget.Ses
					member, err := s.GuildMember(prompt.GuildID, reaction.UserID)
					if err != nil {
						log.Println(err)
						return
					}
					if HasRole(member, prompt.RoleID) {
						err = s.GuildMemberRoleRemove(prompt.GuildID, reaction.UserID, prompt.RoleID)
					} else {
						err = s.GuildMemberRoleAdd(prompt.GuildID, reaction.UserID, prompt.RoleID)
					}
					if err != nil {
						log.Println(err)
					}
				},
			},
			DeleteReactions: true,
		}
		go func() {
			err = w.Hook(session, prompt.ChannelID, prompt.MessageID)
			if err != nil {
				sendMessage(prompt.ChannelID, fmt.Sprintf("Could not hook widget: %s.", err.Error()))
			}
		}()
	}

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
	logErr(session.Close)

}

func onReady(_ *discordgo.Session, _ *discordgo.Ready) {

	// Set the playing status.
	logErr(func() error {
		return session.UpdateStatusComplex(discordgo.UpdateStatusData{
			Game: &discordgo.Game{
				Name: "!alliance",
				Type: 2,
			},
		})
	})
}

func onMessageCreate(_ *discordgo.Session, event *discordgo.MessageCreate) {
	if event.WebhookID != "" {
		return
	}
	if event.Author.Bot {
		return
	}

	if !strings.HasPrefix(event.Content, "IGNORE") {
		var roleId string
		if event.ChannelID == GameNewsQueueId {
			roleId = GameNewsRoleId
		} else if event.ChannelID == ServerNewsQueueId {
			roleId = ServerNewsRoleId
		}
		if roleId != "" {
			roles, _ := session.GuildRoles(event.GuildID)
			var role *discordgo.Role
			for _, r := range roles {
				if r.ID == roleId {
					role = r
				}
			}
			if role == nil {
				return
			}

			role, err := session.GuildRoleEdit(event.GuildID, role.ID, role.Name, role.Color, role.Hoist, role.Permissions, true)
			if err != nil {
				sendMessage(event.ChannelID, err.Error())
				return
			}
			_, err = session.ChannelMessageSend(NewsChannelId, fmt.Sprintf("%s\n\n%s", role.Mention(), event.Content))
			if err != nil {
				sendMessage(event.ChannelID, err.Error())
				return
			}
			_, err = session.GuildRoleEdit(event.GuildID, role.ID, role.Name, role.Color, role.Hoist, role.Permissions, false)
			if err != nil {
				sendMessage(event.ChannelID, err.Error())
				return
			}
		}
	}

	// is command?
	if strings.HasPrefix(strings.TrimSpace(event.Content), *fPrefix) {
		commands := strings.Split(strings.TrimPrefix(strings.TrimSpace(event.Content), *fPrefix), " ")
		if len(commands) == 0 {
			sendMessage(event.ChannelID, "invalid command.")
			return
		}

		switch commands[0] {
		case "role":
			handleRoleCommand(event, commands[1:])
			break
		case "alliance":
			handleAllianceCommand(event, commands[1:])
			break
		}

	}

}

func onGuildLeave(_ *discordgo.Session, event *discordgo.GuildMemberRemove) {
	if event.Roles != nil {
		if HasRole(event.Member, LeaderRoleId) {
			sendMessage(ChannelAdmin,
				fmt.Sprintf("[warning]: Representative %s has left the discord guild. Please assign a new representative manually.", event.Nick))
		}
	}
}

func sendMessage(channelId string, message string) {
	_, err := session.ChannelMessageSend(channelId, message)
	if err != nil {
		log.Println(err)
	}
	return
}

func handleAllianceCommand(event *discordgo.MessageCreate, args []string) {
	if event.ChannelID != CrawAssignmentChannelId {
		return
	}

	if len(args) < 1 {
		sendMessage(event.ChannelID, `usage: `+"```"+`
!alliance <tag> -- join an alliance
!alliance leave -- leave an alliance
!alliance setnick <nick> -- set your own nickname
!alliance promote @username -- promote another user
!alliance demote @username -- demote (or kick) another user`+"```")
		return
	}

	if IsDM(event.ChannelID) {
		sendMessage(event.ChannelID, "Please run this command on a server.")
		return
	}

	switch args[0] {
	case "leave":
		leaveAlliance(event)
		break
	case "setnick":
		if len(args) != 2 {
			sendMessage(event.ChannelID, "Please specify your nickname")
			return
		}
		setnick(event, args[1])
		break
	case "demote":
		if len(event.Mentions) == 0 {
			sendMessage(event.ChannelID, "Please mention who you want to promote.")
			return
		}
		demote(event)
		break
	case "promote":
		if len(event.Mentions) == 0 {
			sendMessage(event.ChannelID, "Please mention who you want to promote.")
			return
		}
		promote(event)
		break
	default:
		if len(args) == 0 {
			sendMessage(event.ChannelID, "invalid command. usage: !alliance <tag>")
			return
		}
		if utf8.RuneCountInString(args[0]) > 4 {
			sendMessage(event.ChannelID, "your alliance tag must be at most four characters")
			return
		}
		if len(args) == 1 {
			joinAlliance(event, args[0], event.Author.Username)
			return
		}
		if len(args) > 1 {
			sendMessage(event.ChannelID, "invalid command. usage: !alliance <tag>")
			return
		}
	}
}

func handleRoleCommand(event *discordgo.MessageCreate, args []string) {
	if event.Author.ID != BotOwnerId {
		return
	}

	if len(args) < 2 {
		sendMessage(event.ChannelID, `usage: `+"```"+`
!role prompt create <channel> <role> <message> -- prompt for a role
!role prompt delete <id> -- delete a role prompt
!role prompt update <id> <message>
`+"```")
		return
	}

	if IsDM(event.ChannelID) {
		sendMessage(event.ChannelID, "this can only be run on a server.")
		return
	}

	switch args[1] {
	case "create":
		if len(args) < 5 {
			sendMessage(event.ChannelID, `usage: `+"```"+`
!role prompt create <channel> <role> <message> -- prompt for a role
`+"```")
			return
		}

		createRolePrompt(event, args[2], args[3], strings.Join(args[4:], " "))
		break
	}

}

func createRolePrompt(event *discordgo.MessageCreate, channelID, roleID, message string) {
	state, err := GetGuildState(event.GuildID)
	if err != nil {
		sendMessage(event.ChannelID, err.Error())
		return
	}

	roles, err := session.GuildRoles(event.GuildID)
	if err != nil {
		sendMessage(event.ChannelID, err.Error())
		return
	}

	var role discordgo.Role
	for _, r := range roles {
		if r.ID == roleID {
			role = *r
		}
	}

	rp := RolePrompt{
		ChannelID: channelID,
		MessageID: "",
		RoleID:    roleID,
		GuildID:   event.GuildID,
		Emote:     "✅",
	}
	rand.Seed(time.Now().UnixNano())

	bot, _ := session.GuildMember(event.GuildID, session.State.User.ID)
	name := bot.User.Username
	if bot.Nick != "" {
		name = bot.Nick
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Toggle %s", role.Name),
		Description: "Click the below emote to give yourself this role.",
		Color:       int((((rand.Int31n(255) << 8) + rand.Int31n(255)) << 8) + (rand.Int31n(255))),
		Timestamp:   time.Now().Format(time.RFC3339),
		Author:      &discordgo.MessageEmbedAuthor{Name: name},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "s157-bot",
		},
	}
	wg := dgwidgets.NewWidget(session, channelID, embed)
	wg.DeleteReactions = true
	wg.Handle("✅", func(widget *dgwidgets.Widget, reaction *discordgo.MessageReaction) {
		s := widget.Ses
		member, err := s.GuildMember(rp.GuildID, reaction.UserID)
		if err != nil {
			log.Println(err)
			return
		}
		if HasRole(member, rp.RoleID) {
			err = s.GuildMemberRoleRemove(rp.GuildID, reaction.UserID, rp.RoleID)
		} else {
			err = s.GuildMemberRoleAdd(rp.GuildID, reaction.UserID, rp.RoleID)
		}
		if err != nil {
			log.Println(err)
		}
	})
	go wg.Spawn()
	for wg.Message == nil {
		time.Sleep(200 * time.Millisecond)
	} // wait for message to be sent
	rp.MessageID = wg.Message.ID
	state.RolePrompts = append(state.RolePrompts, rp)
	err = SetGuildState(event.GuildID, state)
	if err != nil {
		sendMessage(event.ChannelID, err.Error())
	}
}

func demote(event *discordgo.MessageCreate) {
	// is rep?
	event.Member.GuildID = event.GuildID
	isLeader := HasRole(event.Member, LeaderRoleId)
	if !isLeader {
		sendMessage(event.ChannelID, "You are not the representative of your alliance.")
		return
	}

	if len(event.Mentions) != 1 {
		sendMessage(event.ChannelID, "Please mention exactly one user.")
		return
	}
	target := event.Mentions[0]
	if target.ID == event.Author.ID {
		sendMessage(event.ChannelID, "You cannot demote yourself.")
		return
	}
	targetMember, _ := session.GuildMember(event.GuildID, target.ID)
	if !HasRole(targetMember, MemberRoleId) {
		sendMessage(event.ChannelID, "Your target is not in an alliance.")
		return
	}

	if !strings.HasPrefix(targetMember.Nick, strings.Split(event.Member.Nick, " ")[0]) {
		sendMessage(event.ChannelID, "Your target is not in the same alliance as you.")
		return
	}

	if HasRole(targetMember, CommodoreRoleId) {
		err := session.GuildMemberRoleRemove(event.GuildID, target.ID, CommodoreRoleId)
		if err != nil {
			sendMessage(event.ChannelID, "Could not demote user. No permission to remove role.")
		} else {
			sendMessage(event.ChannelID, "User Demoted.")
		}
	} else {

		e := &discordgo.MessageEmbed{
			Title:       "Authorising...",
			Description: fmt.Sprintf("Are you sure you want to kick %s?", targetMember.Nick),
		}

		w := dgwidgets.NewWidget(session, event.ChannelID, e)
		w.DeleteReactions = true
		w.Timeout = 1 * time.Minute
		w.DeleteOnTimeout = true
		w.UserWhitelist = []string{event.Author.ID}

		logErr(func() error {
			return w.Handle("✅", func(widget *dgwidgets.Widget, reaction *discordgo.MessageReaction) {
				// adding new user to alliance
				logErr(func() error {
					return session.GuildMemberNickname(event.GuildID, event.Mentions[0].ID, strings.Split(event.Mentions[0].Username, " ")[1])
				})

				logErr(func() error {
					return session.GuildMemberRoleRemove(event.GuildID, event.Mentions[0].ID, MemberRoleId)
				})
				sendMessage(event.ChannelID, fmt.Sprintf("Kicked %s.", targetMember.Nick))
			})
		})

		logErr(func() error {
			return w.Handle("❌", func(widget *dgwidgets.Widget, reaction *discordgo.MessageReaction) {

				logErr(func() error {
					return session.ChannelMessageDelete(reaction.ChannelID, reaction.MessageID)
				})
				sendMessage(event.ChannelID, "Cancelled that.")
			})
		})

		err := w.Spawn()
		if err != nil && err == context.DeadlineExceeded {
			sendMessage(event.ChannelID, "Timed out.")
		}
	}

}

func setnick(event *discordgo.MessageCreate, nick string) {
	var err error
	if !HasRole(event.Member, MemberRoleId) {
		err = session.GuildMemberNickname(event.GuildID, event.Author.ID, nick)
	} else {
		tag := strings.Split(event.Member.Nick, " ")[0]
		err = session.GuildMemberNickname(event.GuildID, event.Author.ID, tag+" "+nick)
	}
	if err != nil {
		sendMessage(event.ChannelID, "Could not set nickname. Unhandled exception.")
	}
}

func promote(event *discordgo.MessageCreate) {
	// is rep?
	event.Member.GuildID = event.GuildID
	isLeader := HasRole(event.Member, LeaderRoleId)
	if !isLeader {
		sendMessage(event.ChannelID, "You are not the representative of your alliance.")
		return
	}

	if len(event.Mentions) != 1 {
		sendMessage(event.ChannelID, "Please mention exactly one user.")
		return
	}

	target := event.Mentions[0]
	if target.ID == event.Author.ID {
		sendMessage(event.ChannelID, "You cannot promote yourself.")
		return
	}
	targetMember, _ := session.GuildMember(event.GuildID, target.ID)
	if !HasRole(targetMember, MemberRoleId) {
		sendMessage(event.ChannelID, "Your target is not in an alliance.")
		return
	}

	if !strings.HasPrefix(targetMember.Nick, strings.Split(event.Member.Nick, " ")[0]) {
		sendMessage(event.ChannelID, "Your target is not in the same alliance as you.")
		return
	}

	if HasRole(targetMember, CommodoreRoleId) {

		e := &discordgo.MessageEmbed{
			Title:       "Authorising...",
			Description: fmt.Sprintf("Are you sure you want to make %s your successor?", targetMember.Nick),
		}

		w := dgwidgets.NewWidget(session, event.ChannelID, e)
		w.DeleteReactions = true
		w.Timeout = time.Minute
		w.UserWhitelist = []string{event.Author.ID}

		logErr(func() error {
			return w.Handle("✅", func(widget *dgwidgets.Widget, reaction *discordgo.MessageReaction) {
				// adding new user to alliance
				logErr(func() error {
					return session.GuildMemberRoleAdd(event.GuildID, event.Mentions[0].ID, LeaderRoleId)
				})

				logErr(func() error {
					return session.GuildMemberRoleRemove(event.GuildID, event.Mentions[0].ID, CommodoreRoleId)
				})

				logErr(func() error {
					return session.GuildMemberRoleRemove(event.GuildID, event.Author.ID, LeaderRoleId)
				})
				sendMessage(event.ChannelID, fmt.Sprintf("Made %s the new alliance leader.", targetMember.Nick))
			})
		})

		logErr(func() error {
			return w.Handle("❌", func(widget *dgwidgets.Widget, reaction *discordgo.MessageReaction) {

				logErr(func() error {
					return session.ChannelMessageDelete(reaction.ChannelID, reaction.MessageID)
				})
				sendMessage(event.ChannelID, "Cancelled that.")
			})
		})

		logErr(w.Spawn)
	} else {

		logErr(func() error {
			return session.GuildMemberRoleAdd(event.GuildID, event.Mentions[0].ID, CommodoreRoleId)
		})
		sendMessage(event.ChannelID, fmt.Sprintf("Made %s a commodore.", targetMember.Nick))
	}

}

func HasRole(member *discordgo.Member, id string) bool {
	for _, role := range member.Roles {
		if role == id {
			return true
		}
	}
	return false
}

type PendingRequest struct {
	ChannelID        string
	RepresentativeID string
	TargetID         string
	GuildID          string
	AllianceTag      string
	MessageID        string
	CreatedAt        time.Time
}

type RolePrompt struct {
	ChannelID string
	MessageID string
	RoleID    string
	GuildID   string
	Emote     string
}

type GuildState struct {
	PendingRequests []PendingRequest
	RolePrompts     []RolePrompt
}

func GetGuildState(guild string) (state GuildState, err error) {
	msg, err := FindStateMessage(guild)
	if err != nil { // no state message
		return GuildState{}, nil
	}
	err = json.Unmarshal([]byte(msg.Content), &state)
	return
}

func SetGuildState(guild string, state GuildState) error {
	b, err := json.Marshal(state)
	if err != nil {
		return err
	}
	msg, err := FindStateMessage(guild)
	if err != nil || msg.Author.ID != session.State.User.ID {
		// create new message
		if msg == nil {
			return err
		}
		_, err := session.ChannelMessageSend(msg.ChannelID, string(b))
		return err
	} else {
		// update existing message
		_, err := session.ChannelMessageEdit(msg.ChannelID, msg.ID, string(b))
		return err
	}
}

func FindStateMessage(guild string) (*discordgo.Message, error) {
	channel, err := FindChannel(guild, "_data")
	if err != nil {
		return nil, err
	}
	msg, err := session.ChannelMessage(channel.ID, channel.LastMessageID)
	if err != nil {
		return &discordgo.Message{ChannelID: channel.ID}, ErrNoData
	}
	return msg, nil
}

func FindChannel(guild, name string) (*discordgo.Channel, error) {
	channels, err := session.GuildChannels(guild)
	if err != nil {
		return nil, err
	}
	for _, c := range channels {
		if c.Name == name {
			return c, nil
		}
	}
	return session.GuildChannelCreate(guild, "_data", discordgo.ChannelTypeGuildText)
}

// if the message is a private message, error
// if a user is not in an alliance, error
// if a user is a representative, error and have them choose a different rep
// if a user is the only alliance member, remove any tags
// else remove any tags
func leaveAlliance(event *discordgo.MessageCreate) {
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
	if !HasRole(event.Member, MemberRoleId) {
		_, err = session.ChannelMessageSend(channel.ID, "You are not a member of any alliance.")
		return
	}
	if HasRole(event.Member, LeaderRoleId) {
		members := GetMembers(event.GuildID)
		m := make([]*discordgo.Member, 0)
		for _, member := range members {
			for _, r := range member.Roles {
				if r == MemberRoleId && strings.HasPrefix(member.Nick, strings.Split(event.Member.Nick, " ")[0]) && member.User.ID != event.Author.ID {
					m = append(m, member)
				}
			}
		}
		if len(m) > 0 {

			sendMessage(channel.ID, "Your current alliance has members. Promote someone before leaving.")
			return
		}

		logErr(func() error {
			return session.GuildMemberRoleRemove(event.GuildID, event.Author.ID, LeaderRoleId)
		})
	}

	logErr(func() error {
		return session.GuildMemberRoleRemove(event.GuildID, event.Author.ID, MemberRoleId)
	})
	err = session.GuildMemberNickname(event.GuildID, event.Author.ID, strings.Split(event.Member.Nick, "] ")[1])
	if err != nil {
		sendMessage(event.ChannelID, "couldn't change your nickname.")
	}
}

func joinAlliance(event *discordgo.MessageCreate, tag, user string) {
	tag = strings.ToUpper(tag)
	if IsDM(event.ChannelID) {
		sendMessage(event.ChannelID, "Please use this command on a server.")
		return
	}
	if HasRole(event.Member, MemberRoleId) {
		sendMessage(event.ChannelID, "You are already in an alliance.")
		return
	}

	// check for members in same guild
	guildMembers := GetMembers(event.GuildID)
	var leader *discordgo.Member
	// if none - add and make rep
	for _, member := range guildMembers {
		for _, role := range member.Roles {
			if role == LeaderRoleId {
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
			sendMessage(event.ChannelID, "couldn't change your nickname. aborting.")
			log.Println(err)
			return
		}
		err = session.GuildMemberRoleAdd(event.GuildID, event.Author.ID, MemberRoleId)
		if err != nil {
			log.Println(err)
			sendMessage(event.ChannelID, "couldn't give you the alliance role.")
		}
		err = session.GuildMemberRoleAdd(event.GuildID, event.Author.ID, LeaderRoleId)
		if err != nil {
			log.Println(err)
			sendMessage(event.ChannelID, "couldn't give you the alliance leader role.")
		}
		if HasRole(event.Member, NameNotChanged) {
			err = session.GuildMemberRoleRemove(event.GuildID, event.Author.ID, NameNotChanged)
			if err != nil {
				sendMessage(event.ChannelID, "Could not remove redundant role.")
			}
		}
		sendMessage(event.ChannelID, "You are the first person to join this guild. Welcome, admiral.")
	} else {
		sendMessage(event.ChannelID, "asking your representative for permission")
		channel, err := session.UserChannelCreate(leader.User.ID)
		if err != nil {
			sendMessage(event.ChannelID, "couldn't ask your rep. please contact an admin.")
			return
		}
		e := &discordgo.MessageEmbed{
			Title:       "Someone wants to join your alliance!",
			Description: fmt.Sprintf("%s wants to join your alliance.", event.Author.Username),
		}

		w := dgwidgets.NewWidget(session, channel.ID, e)

		logErr(func() error {
			return w.Handle("✅", func(widget *dgwidgets.Widget, reaction *discordgo.MessageReaction) {
				// adding new user to alliance

				logErr(func() error {
					return session.GuildMemberNickname(event.GuildID, event.Author.ID, "["+tag+"] "+user)
				})

				logErr(func() error {
					return session.GuildMemberRoleAdd(event.GuildID, event.Author.ID, MemberRoleId)
				})
				if HasRole(event.Member, NameNotChanged) {
					err = session.GuildMemberRoleRemove(event.GuildID, event.Author.ID, NameNotChanged)
					if err != nil {
						sendMessage(event.ChannelID, "Could not remove role.")
					}
				}
			})
		})

		logErr(func() error {
			return w.Handle("❌", func(widget *dgwidgets.Widget, reaction *discordgo.MessageReaction) {
				logErr(func() error {
					return session.ChannelMessageDelete(reaction.ChannelID, reaction.MessageID)
				})
			})
		})
		logErr(w.Spawn)

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

func logErr(some func() error) {
	if err := some(); err != nil {
		log.Println(err)
	}
}
