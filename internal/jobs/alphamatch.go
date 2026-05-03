package jobs

//Alpha match is the very first matching algorithm
//Just will form groups based on # of players and timestamp. will essentially ignore elo rating at start
import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// AlphaMatchJob runs every 2 minutes.
type AlphaMatchJob struct{}

type player struct {
	id       int
	elo      int
	joinedAt int64
}

func (m AlphaMatchJob) Name() string {
	return "AlphaMatchJob"
}

func (m AlphaMatchJob) Schedule() string {
	return "*/10 * * * * *" // Runs every 10 seconds
}

func (m AlphaMatchJob) Run(ctx context.Context, rdb *redis.Client) error {
	select {
	case <-ctx.Done():
		os.Exit(0) // Main thread is done
	default:
	}
	const matchSize = 4

	//Pull from queue:ranked:elo if two players are found then match.
	elements, err := rdb.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     "queue:ranked:elo",
		Start:   "0",
		Stop:    "10",
		ByScore: true,
	}).Result()
	if err != nil {
		log.Print("ERROR: ", err)
	}

	if len(elements) < matchSize {
		//log.Println("not enough players to match, skipping")
		return nil
	}
	pipe := rdb.Pipeline()

	// queue members is []string of userIDs from ZRangeArgs
	cmds := make([]*redis.MapStringStringCmd, len(elements))
	for i, userID := range elements {
		cmds[i] = pipe.HGetAll(ctx, "player:"+userID)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("pipeline HGetAll: %w", err)
	}

	avalPlayers := make([]player, 0, len(elements))
	for i, cmd := range cmds {
		playerData, err := cmd.Result()
		if err != nil {
			continue
		}
		//map playerData to playet
		playerID, idErr := strconv.Atoi(elements[i])
		parsedElo, eloErr := strconv.Atoi(playerData["elo"])
		parsedJoined, joinedErr := strconv.ParseInt(playerData["joined_at"], 10, 64)
		if idErr != nil || eloErr != nil || joinedErr != nil {
			// TODO: handle error better
			log.Fatal("MEGA PARSE ERROR WITH REDIS!")
		}
		tempPlayer := player{id: playerID, elo: parsedElo, joinedAt: parsedJoined}
		avalPlayers = append(avalPlayers, tempPlayer)

	}
	sort.Slice(avalPlayers, func(i, j int) bool {
		return avalPlayers[i].joinedAt < avalPlayers[j].joinedAt
	})

	//matches := make([]player, 0, (len(avalPlayers)%4)+1)
	//Insert to redis match
	for x := 0; x+matchSize <= len(avalPlayers); x += matchSize {
		group := avalPlayers[x : x+matchSize]
		matchID := fmt.Sprintf("match:%d", time.Now().UnixNano())

		// Build team assignments: players 0,1 → team 1 | players 2,3 → team 2
		teamAssignments := make([]interface{}, 0, matchSize*2)
		for i, p := range group {
			teamID := i/2 + 1
			teamAssignments = append(teamAssignments, strconv.Itoa(p.id), teamID)
		}
		log.Println("MATCHID: ", matchID)
		log.Println("Player and Team ", teamAssignments[0], teamAssignments[1])

		matchPipe := rdb.Pipeline()

		matchPipe.HSet(ctx, "match:"+matchID, teamAssignments...)

		//Cleanup
		for _, p := range group {
			userIDStr := strconv.Itoa(p.id)

			//Playerid:id:match:matchId
			matchPipe.Set(ctx, "player:"+userIDStr+":match", matchID, 10*time.Minute)

			matchPipe.ZRem(ctx, "queue:ranked:elo", userIDStr)
			matchPipe.Del(ctx, "player:"+userIDStr)
		}

		if _, err := matchPipe.Exec(ctx); err != nil {
			log.Printf("ERROR: failed to commit match %s: %v", matchID, err)
			continue
		}

		log.Printf("INFO: match %s created with %d players", matchID, matchSize)

	}
	return nil
}
