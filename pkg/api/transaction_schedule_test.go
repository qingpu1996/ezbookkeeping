package api

import (
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/validators"
	"testing"
)

func TestScheduleMinuteCompatibility(t *testing.T) {
	binding.Validator.Engine().(*validator.Validate).RegisterValidation("notBlank", validators.NotBlank)
	for _, tc := range []struct{ local, offset, want int16 }{{557, 480, 77}, {0, 480, 960}, {1439, -720, 719}, {1, 840, 601}} {
		if got := scheduleMinuteUTC(&tc.local, tc.offset, 0); got != tc.want {
			t.Fatalf("%+v got %d", tc, got)
		}
	}
	if scheduleMinuteUTC(nil, 480, 0) != 960 {
		t.Fatal("legacy creation changed")
	}
	if scheduleMinuteUTC(nil, 480, 557) != 77 {
		t.Fatal("legacy editing lost custom time")
	}
	for _, n := range []int16{-1, 1440} {
		req := models.TransactionTemplateCreateRequest{Name: "test", Type: 1, CategoryId: 1, SourceAccountId: 1, ScheduledMinute: &n}
		if binding.Validator.ValidateStruct(req) == nil {
			t.Fatal("invalid minute accepted", n)
		}
	}
}
