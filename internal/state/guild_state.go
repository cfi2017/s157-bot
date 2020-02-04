package state

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	ErrNoData = errors.New("no such error")
)

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
	Prefix          string
}

type key int

var contextKey key

func NewContext(ctx context.Context, session *discordgo.Session) context.Context {
	return context.WithValue(ctx, contextKey, NewGuildStateManager(session))
}

func FromContextOptimistic(ctx context.Context) GuildStateManager {
	return ctx.Value(contextKey).(GuildStateManager)
}

func FromContext(ctx context.Context) (GuildStateManager, bool) {
	m, ok := ctx.Value(contextKey).(GuildStateManager)
	return m, ok
}

func NewGuildStateManager(session *discordgo.Session) *GuildStateManager {
	return &GuildStateManager{
		session: session,
	}
}

type GuildStateManager struct {
	sync.RWMutex
	States  map[string]GuildState
	clean   map[string]bool
	session *discordgo.Session
}

func (m *GuildStateManager) GetGuildState(guild string) (state GuildState, err error) {
	m.RLock()
	if m.clean[guild] {
		defer m.RUnlock()
		return m.States[guild], nil
	}
	m.RUnlock()
	state, err = GetGuildState(m.session, guild)
	if err != nil {
		return
	}
	m.Lock()
	m.States[guild] = state
	m.clean[guild] = true
	m.Unlock()
	return
}

func (m *GuildStateManager) SetGuildState(guild string, state GuildState) error {
	err := SetGuildState(m.session, guild, state)
	if err != nil {
		return err
	}
	m.Lock()
	m.States[guild] = state
	m.clean[guild] = true
	m.Unlock()
	return nil
}

func GetGuildState(session *discordgo.Session, guild string) (state GuildState, err error) {
	msg, err := findStateMessage(session, guild)
	if err != nil { // no state message
		return GuildState{}, nil
	}
	err = json.Unmarshal([]byte(msg.Content), &state)
	return
}

func SetGuildState(session *discordgo.Session, guild string, state GuildState) error {
	b, err := json.Marshal(state)
	if err != nil {
		return err
	}
	msg, err := findStateMessage(session, guild)
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

func findStateMessage(session *discordgo.Session, guild string) (*discordgo.Message, error) {
	channel, err := findChannel(session, guild, "_data")
	if err != nil {
		return nil, err
	}
	msg, err := session.ChannelMessage(channel.ID, channel.LastMessageID)
	if err != nil {
		return &discordgo.Message{ChannelID: channel.ID}, ErrNoData
	}
	return msg, nil
}

func findChannel(session *discordgo.Session, guild, name string) (*discordgo.Channel, error) {
	channels, err := session.GuildChannels(guild)
	if err != nil {
		return nil, err
	}
	for _, c := range channels {
		if c.Name == name {
			return c, nil
		}
	}
	return createChannel(session, guild, name)
}

func createChannel(session *discordgo.Session, guild, name string) (channel *discordgo.Channel, err error) {
	channel, err = session.GuildChannelCreate(guild, name, discordgo.ChannelTypeGuildText)
	if err != nil {
		return
	}
	_ = session.ChannelPermissionSet(channel.ID, guild, "role", 0x0, 0x00000400)
	_ = session.ChannelPermissionSet(channel.ID, session.State.User.ID, "member", 0x00000400, 0x0)
	return
}
