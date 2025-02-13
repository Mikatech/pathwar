package pwdb

import (
	"encoding/json"
	"fmt"
	"strings"
	time "time"

	"github.com/martinlindhe/base36"
	"go.uber.org/zap"
	"golang.org/x/crypto/sha3"
	"pathwar.land/pathwar/v2/go/pkg/errcode"
	"pathwar.land/pathwar/v2/go/pkg/pwinit"
)

func (cf *ChallengeFlavor) addSeasonChallengeByID(seasonID int64) {
	if cf.SeasonChallenges == nil {
		cf.SeasonChallenges = []*SeasonChallenge{}
	}
	cf.SeasonChallenges = append(cf.SeasonChallenges, &SeasonChallenge{
		SeasonID: seasonID,
	})
}

func (a *Agent) TagSlice() []string {
	if a.Tags == "" {
		return nil
	}
	return strings.Split(a.Tags, ", ")
}

func (cf ChallengeFlavor) NameAndVersion() string {
	return fmt.Sprintf("%s@%s", cf.Challenge.Name, cf.Version)
}

func (instance *ChallengeInstance) ParseInstanceConfig() (*pwinit.InitConfig, error) {
	var configData pwinit.InitConfig
	err := json.Unmarshal(instance.GetInstanceConfig(), &configData)
	if err != nil {
		return nil, errcode.ErrParseInitConfig.Wrap(err)
	}
	return &configData, nil
}

func ChallengeInstancePrefixHash(instanceID string, userID int64, salt string) (string, error) {
	stringToHash := fmt.Sprintf("%s%d%s", instanceID, userID, salt)
	hashBytes := make([]byte, 8)
	hasher := sha3.NewShake256()
	_, err := hasher.Write([]byte(stringToHash))
	if err != nil {
		return "", errcode.ErrWriteBytesToHashBuilder.Wrap(err)
	}
	_, err = hasher.Read(hashBytes)
	if err != nil {
		return "", errcode.ErrReadBytesFromHashBuilder.Wrap(err)
	}
	userHash := strings.ToLower(base36.EncodeBytes(hashBytes))[:8] // we voluntarily expect short hashes here
	return userHash, nil
}

func (m *SeasonChallenge) GetActiveSubscriptions() []*ChallengeSubscription {
	cs := make([]*ChallengeSubscription, 0)

	for _, subscription := range m.GetSubscriptions() {
		if subscription.GetStatus() == ChallengeSubscription_Active {
			cs = append(cs, subscription)
		}
	}

	return cs
}

type constantClock time.Time

func (c constantClock) Now() time.Time                         { return time.Time(c) }
func (c constantClock) NewTicker(d time.Duration) *time.Ticker { return &time.Ticker{} }

func (a *Activity) Log(logger *zap.Logger) {
	var (
		level   = zap.InfoLevel
		inst    = logger.WithOptions(zap.WithClock(constantClock(*a.CreatedAt)))
		message = fmt.Sprintf("%-30s", a.GetKind().String())
	)

	if author := a.GetAuthor(); author != nil {
		inst = inst.With(
			zap.String("author", author.GetSlug()),
		)
		if user := a.GetUser(); user != nil {
			if user.GetID() != author.GetID() {
				inst = inst.With(
					zap.String("user", user.GetSlug()),
				)
			}
		}
	}
	if agent := a.GetAgent(); agent != nil {
		inst = inst.With(
			zap.String("agent", agent.GetSlug()),
		)
	}
	if season := a.GetSeason(); season != nil {
		inst = inst.With(
			zap.String("season", season.GetSlug()),
		)
		if !season.IsGlobal {
			if team := a.GetTeam(); team != nil {
				inst = inst.With(
					zap.String("team", team.GetSlug()),
				)
			}
		}
	}
	if seasonChallenge := a.GetSeasonChallenge(); seasonChallenge != nil {
		inst = inst.With(
			zap.String("season-challenge", seasonChallenge.GetSlug()),
		)
	}
	// FIXME: support more fields
	inst = inst.With(zap.Int("activity", int(a.GetID())))

	switch level {
	case zap.DebugLevel:
		inst.Debug(message)
	case zap.InfoLevel:
		inst.Info(message)
	case zap.WarnLevel:
		inst.Warn(message)
	case zap.ErrorLevel:
		inst.Error(message)
	default:
		panic("invalid level")
	}
}
