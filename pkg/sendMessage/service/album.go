package send_service

import (
	"crypto/rand"
	"errors"
	"strings"

	instance_model "github.com/EvolutionAPI/evolution-go/pkg/instance/model"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

// AlbumStruct starts a native album. Send its children through /send/media
// with the returned message ID and their zero-based albumIndex.
type AlbumStruct struct {
	Number     string `json:"number"`
	ImageCount uint32 `json:"imageCount"`
	VideoCount uint32 `json:"videoCount"`
	Delay      int32  `json:"delay"`
	FormatJid  *bool  `json:"formatJid,omitempty"`
}

func (data *AlbumStruct) Validate() error {
	if data.Number == "" || strings.Contains(data.Number, "@newsletter") {
		return errors.New("albums require a chat or group recipient")
	}
	count := uint64(data.ImageCount) + uint64(data.VideoCount)
	if count < 2 || count > 30 {
		return errors.New("albums require between 2 and 30 images or videos")
	}
	if data.Delay < 0 {
		return errors.New("delay must not be negative")
	}
	return nil
}

func (s *sendService) SendAlbum(data *AlbumStruct, instance *instance_model.Instance) (*MessageSendStruct, error) {
	if err := data.Validate(); err != nil {
		return nil, err
	}
	if _, err := s.ensureClientConnectedWithRetry(instance.Id, 2); err != nil {
		return nil, err
	}
	return s.SendMessage(instance, &waE2E.Message{
		AlbumMessage: &waE2E.AlbumMessage{
			ExpectedImageCount: proto.Uint32(data.ImageCount),
			ExpectedVideoCount: proto.Uint32(data.VideoCount),
		},
	}, "AlbumMessage", &SendDataStruct{
		Number:    data.Number,
		Delay:     data.Delay,
		FormatJid: data.FormatJid,
	})
}

func attachAlbumMedia(message *waE2E.Message, data *MediaStruct) error {
	if data.AlbumID == "" {
		return nil
	}
	if (data.Type != "image" && data.Type != "video") || data.AlbumIndex < 0 || strings.Contains(data.Number, "@newsletter") {
		return errors.New("album children must be images or videos with a non-negative index in a chat or group")
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return err
	}
	message.MessageContextInfo = &waE2E.MessageContextInfo{
		MessageSecret: secret,
		MessageAssociation: &waE2E.MessageAssociation{
			AssociationType: waE2E.MessageAssociation_MEDIA_ALBUM.Enum(),
			ParentMessageKey: &waCommon.MessageKey{
				FromMe:    proto.Bool(true),
				ID:        proto.String(data.AlbumID),
				RemoteJID: proto.String(data.Number),
			},
			MessageIndex: proto.Int32(data.AlbumIndex),
		},
	}
	return nil
}
