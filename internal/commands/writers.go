package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/Minh-Tam-Solution/MTClaw/internal/bus"
	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
	pkgprotocol "github.com/Minh-Tam-Solution/MTClaw/pkg/protocol"
)

// WritersCmd handles /writers, /addwriter, /removewriter commands.
type WritersCmd struct {
	AgentStore store.AgentStore
	Bus        *bus.MessageBus
}

// ListWriters returns a formatted list of file writers for a group.
func (w *WritersCmd) ListWriters(ctx context.Context, agentKey, channelName, chatID string) (string, error) {
	agentUUID, err := ResolveAgentUUID(ctx, w.AgentStore, agentKey)
	if err != nil {
		return "", fmt.Errorf("could not resolve agent: %w", err)
	}

	groupID := fmt.Sprintf("group:%s:%s", channelName, chatID)

	writers, err := w.AgentStore.ListGroupFileWriters(ctx, agentUUID, groupID)
	if err != nil {
		return "", fmt.Errorf("failed to list writers: %w", err)
	}

	if len(writers) == 0 {
		return "No file writers configured for this group. The first person to interact with the bot will be added automatically.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("File writers for this group (%d):\n", len(writers)))
	for i, wr := range writers {
		label := wr.UserID
		if wr.Username != nil && *wr.Username != "" {
			label = "@" + *wr.Username
		} else if wr.DisplayName != nil && *wr.DisplayName != "" {
			label = *wr.DisplayName
		}
		sb.WriteString(fmt.Sprintf("%d. %s (ID: %s)\n", i+1, label, wr.UserID))
	}
	return sb.String(), nil
}

// AddWriter adds a user as a file writer for a group.
func (w *WritersCmd) AddWriter(ctx context.Context, agentKey, channelName, chatID, senderID, targetID, targetDisplayName, targetUsername string) (string, error) {
	agentUUID, err := ResolveAgentUUID(ctx, w.AgentStore, agentKey)
	if err != nil {
		return "", fmt.Errorf("could not resolve agent: %w", err)
	}

	groupID := fmt.Sprintf("group:%s:%s", channelName, chatID)
	senderNumericID := strings.SplitN(senderID, "|", 2)[0]

	// Check sender is an existing writer
	isWriter, err := w.AgentStore.IsGroupFileWriter(ctx, agentUUID, groupID, senderNumericID)
	if err != nil {
		return "", fmt.Errorf("failed to check permissions: %w", err)
	}
	if !isWriter {
		return "", fmt.Errorf("only existing file writers can manage the writer list")
	}

	if err := w.AgentStore.AddGroupFileWriter(ctx, agentUUID, groupID, targetID, targetDisplayName, targetUsername); err != nil {
		return "", fmt.Errorf("failed to add writer: %w", err)
	}

	targetName := targetDisplayName
	if targetUsername != "" {
		targetName = "@" + targetUsername
	}

	w.Bus.Broadcast(bus.Event{
		Name:    pkgprotocol.EventCacheInvalidate,
		Payload: bus.CacheInvalidatePayload{Kind: bus.CacheKindGroupFileWriters, Key: groupID},
	})

	return fmt.Sprintf("Added %s as a file writer.", targetName), nil
}

// RemoveWriter removes a user from the file writer list for a group.
func (w *WritersCmd) RemoveWriter(ctx context.Context, agentKey, channelName, chatID, senderID, targetID, targetDisplayName, targetUsername string) (string, error) {
	agentUUID, err := ResolveAgentUUID(ctx, w.AgentStore, agentKey)
	if err != nil {
		return "", fmt.Errorf("could not resolve agent: %w", err)
	}

	groupID := fmt.Sprintf("group:%s:%s", channelName, chatID)
	senderNumericID := strings.SplitN(senderID, "|", 2)[0]

	// Check sender is an existing writer
	isWriter, err := w.AgentStore.IsGroupFileWriter(ctx, agentUUID, groupID, senderNumericID)
	if err != nil {
		return "", fmt.Errorf("failed to check permissions: %w", err)
	}
	if !isWriter {
		return "", fmt.Errorf("only existing file writers can manage the writer list")
	}

	// Prevent removing the last writer
	writers, _ := w.AgentStore.ListGroupFileWriters(ctx, agentUUID, groupID)
	if len(writers) <= 1 {
		return "", fmt.Errorf("cannot remove the last file writer")
	}

	if err := w.AgentStore.RemoveGroupFileWriter(ctx, agentUUID, groupID, targetID); err != nil {
		return "", fmt.Errorf("failed to remove writer: %w", err)
	}

	targetName := targetDisplayName
	if targetUsername != "" {
		targetName = "@" + targetUsername
	}

	w.Bus.Broadcast(bus.Event{
		Name:    pkgprotocol.EventCacheInvalidate,
		Payload: bus.CacheInvalidatePayload{Kind: bus.CacheKindGroupFileWriters, Key: groupID},
	})

	return fmt.Sprintf("Removed %s from file writers.", targetName), nil
}
