package send_service

import (
	"testing"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func TestAlbumValidation(t *testing.T) {
	for _, data := range []AlbumStruct{
		{Number: "120@g.us", ImageCount: 2},
		{Number: "120@g.us", ImageCount: 1, VideoCount: 1},
		{Number: "120@g.us", ImageCount: 30},
	} {
		if err := data.Validate(); err != nil {
			t.Fatalf("valid album rejected: %v", err)
		}
	}
	for _, data := range []AlbumStruct{
		{ImageCount: 2},
		{Number: "120@newsletter", ImageCount: 2},
		{Number: "120@g.us", ImageCount: 1},
		{Number: "120@g.us", ImageCount: 31},
		{Number: "120@g.us", ImageCount: ^uint32(0), VideoCount: 3},
		{Number: "120@g.us", ImageCount: 2, Delay: -1},
	} {
		if data.Validate() == nil {
			t.Fatalf("invalid album accepted: %+v", data)
		}
	}
}

func TestAlbumMediaAssociationPreservesCaption(t *testing.T) {
	message := &waE2E.Message{ImageMessage: &waE2E.ImageMessage{Caption: proto.String("ofertas")}}
	err := attachAlbumMedia(message, &MediaStruct{Number: "120@g.us", Type: "image", AlbumID: "PARENT", AlbumIndex: 1})
	if err != nil {
		t.Fatal(err)
	}
	association := message.GetMessageContextInfo().GetMessageAssociation()
	if association.GetAssociationType() != waE2E.MessageAssociation_MEDIA_ALBUM || association.GetMessageIndex() != 1 {
		t.Fatal("missing album association or wrong media index")
	}
	key := association.GetParentMessageKey()
	if key.GetID() != "PARENT" || key.GetRemoteJID() != "120@g.us" || !key.GetFromMe() {
		t.Fatalf("incorrect parent key: %v", key)
	}
	if len(message.GetMessageContextInfo().GetMessageSecret()) != 32 || message.GetImageMessage().GetCaption() != "ofertas" {
		t.Fatal("missing secret or lost caption")
	}
}

func TestOrdinaryMediaDoesNotBecomeAlbumChild(t *testing.T) {
	message := &waE2E.Message{ImageMessage: &waE2E.ImageMessage{}}
	if err := attachAlbumMedia(message, &MediaStruct{Type: "image"}); err != nil || message.MessageContextInfo != nil {
		t.Fatal("ordinary media was changed")
	}
}

func TestInvalidAlbumChildrenAreRejected(t *testing.T) {
	for _, data := range []MediaStruct{
		{Type: "audio", AlbumID: "PARENT"},
		{Type: "document", AlbumID: "PARENT"},
		{Type: "image", AlbumID: "PARENT", AlbumIndex: -1},
		{Type: "image", AlbumID: "PARENT", Number: "120@newsletter"},
	} {
		if attachAlbumMedia(&waE2E.Message{}, &data) == nil {
			t.Fatalf("invalid album child accepted: %+v", data)
		}
	}
}
