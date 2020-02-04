package internal

import "github.com/bwmarrin/discordgo"

func IsDM(session *discordgo.Session, channel string) bool {
	c, err := session.Channel(channel)
	if err != nil {
		return false
	}
	return c.Type == discordgo.ChannelTypeDM
}

func HasRole(member *discordgo.Member, id string) bool {
	for _, role := range member.Roles {
		if role == id {
			return true
		}
	}
	return false
}

func GetMembers(session *discordgo.Session, guild string) []*discordgo.Member {
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

func IsChannelInGuild(session *discordgo.Session, guildId, channelId string) (bool, error) {
	g, err := session.Guild(guildId)
	if err != nil {
		return false, err
	}
	for _, channel := range g.Channels {
		if channel.ID == channelId {
			return true, nil
		}
	}
	return false, nil
}
