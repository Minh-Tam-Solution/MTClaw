package commands

import (
	"strings"

	"github.com/Minh-Tam-Solution/MTClaw/internal/bus"
)

// PublishReset publishes a /reset command to the bus.
func PublishReset(msgBus *bus.MessageBus, channel, senderID, chatID, agentID, peerKind string, meta CommandMetadata) {
	meta.Command = "reset"
	msgBus.PublishInbound(bus.InboundMessage{
		Channel:  channel,
		SenderID: senderID,
		ChatID:   chatID,
		Content:  "/reset",
		PeerKind: peerKind,
		AgentID:  agentID,
		UserID:   strings.SplitN(senderID, "|", 2)[0],
		Metadata: meta.ToMap(),
	})
}

// PublishStop publishes a /stop command to the bus.
func PublishStop(msgBus *bus.MessageBus, channel, senderID, chatID, agentID, peerKind string, meta CommandMetadata) {
	meta.Command = "stop"
	msgBus.PublishInbound(bus.InboundMessage{
		Channel:  channel,
		SenderID: senderID,
		ChatID:   chatID,
		Content:  "/stop",
		PeerKind: peerKind,
		AgentID:  agentID,
		UserID:   strings.SplitN(senderID, "|", 2)[0],
		Metadata: meta.ToMap(),
	})
}

// PublishStopAll publishes a /stopall command to the bus.
func PublishStopAll(msgBus *bus.MessageBus, channel, senderID, chatID, agentID, peerKind string, meta CommandMetadata) {
	meta.Command = "stopall"
	msgBus.PublishInbound(bus.InboundMessage{
		Channel:  channel,
		SenderID: senderID,
		ChatID:   chatID,
		Content:  "/stopall",
		PeerKind: peerKind,
		AgentID:  agentID,
		UserID:   strings.SplitN(senderID, "|", 2)[0],
		Metadata: meta.ToMap(),
	})
}

// PublishSpec publishes a /spec command to the bus, routing to the PM SOUL.
func PublishSpec(msgBus *bus.MessageBus, channel, senderID, chatID, peerKind, taskText string, meta CommandMetadata) {
	meta.Command = "spec"
	meta.Rail = "spec-factory"
	msgBus.PublishInbound(bus.InboundMessage{
		Channel:  channel,
		SenderID: senderID,
		ChatID:   chatID,
		Content:  taskText,
		PeerKind: peerKind,
		AgentID:  "pm",
		UserID:   strings.SplitN(senderID, "|", 2)[0],
		Metadata: meta.ToMap(),
	})
}

// PublishReview publishes a /review command to the bus, routing to the Reviewer SOUL.
func PublishReview(msgBus *bus.MessageBus, channel, senderID, chatID, peerKind, prURL string, meta CommandMetadata) {
	meta.Command = "review"
	meta.Rail = "pr-gate"
	meta.PRURL = prURL
	msgBus.PublishInbound(bus.InboundMessage{
		Channel:  channel,
		SenderID: senderID,
		ChatID:   chatID,
		Content:  prURL,
		PeerKind: peerKind,
		AgentID:  "reviewer",
		UserID:   strings.SplitN(senderID, "|", 2)[0],
		Metadata: meta.ToMap(),
	})
}
