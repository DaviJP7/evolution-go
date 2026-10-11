package send_handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	instance_model "github.com/EvolutionAPI/evolution-go/pkg/instance/model"
	send_service "github.com/EvolutionAPI/evolution-go/pkg/sendMessage/service"
	"github.com/gin-gonic/gin"
)

type albumServiceStub struct {
	send_service.SendService
	calls int
	err   error
}

func (s *albumServiceStub) SendAlbum(data *send_service.AlbumStruct, instance *instance_model.Instance) (*send_service.MessageSendStruct, error) {
	s.calls++
	return &send_service.MessageSendStruct{}, s.err
}

func (s *albumServiceStub) SendMediaUrl(data *send_service.MediaStruct, instance *instance_model.Instance) (*send_service.MessageSendStruct, error) {
	s.calls++
	return nil, s.err
}

func TestAlbumHandlerValidatesBeforeSending(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &albumServiceStub{}
	handler := &sendHandler{sendMessageService: stub}
	router := gin.New()
	router.Use(func(ctx *gin.Context) { ctx.Set("instance", &instance_model.Instance{}) })
	router.POST("/send/album", handler.SendAlbum)
	for _, body := range []string{`null`, `{}`, `{"number":"120@g.us","imageCount":1}`} {
		request := httptest.NewRequest(http.MethodPost, "/send/album", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || stub.calls != 0 {
			t.Fatalf("invalid input reached send service: %d %s", response.Code, response.Body.String())
		}
	}
	request := httptest.NewRequest(http.MethodPost, "/send/album", strings.NewReader(`{"number":"120@g.us","imageCount":2}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || stub.calls != 1 {
		t.Fatalf("valid album was not sent: %d %s", response.Code, response.Body.String())
	}
}

func TestAlbumSendFailuresAreAmbiguous(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &albumServiceStub{err: errors.New("lost acknowledgement")}
	handler := &sendHandler{sendMessageService: stub}
	router := gin.New()
	router.Use(func(ctx *gin.Context) { ctx.Set("instance", &instance_model.Instance{}) })
	router.POST("/send/album", handler.SendAlbum)
	router.POST("/send/media", handler.SendMedia)
	for path, body := range map[string]string{
		"/send/album": `{"number":"120@g.us","imageCount":2}`,
		"/send/media": `{"number":"120@g.us","type":"image","url":"https://cdn.test/1.jpg","albumId":"PARENT"}`,
	} {
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusBadGateway || !strings.Contains(response.Body.String(), `"code":"ambiguous_delivery"`) {
			t.Fatalf("album delivery failure could be retried unsafely: %d %s", response.Code, response.Body.String())
		}
	}
}
