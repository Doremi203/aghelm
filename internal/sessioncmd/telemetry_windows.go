//go:build windows

package sessioncmd

import "github.com/Doremi203/aghelm/internal/conversation"

// agentLiveness tells the follower when the agent is gone, or that it
// already is. Windows has no process group, so the agent's own pid is what
// has to exit.
func agentLiveness(agent int) (alive func(int) bool, gone bool, err error) {
	if !conversation.Alive(agent) {
		return nil, true, nil
	}
	return conversation.Alive, false, nil
}
