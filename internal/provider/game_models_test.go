package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// helpers

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func cardDisplayMode(s string) *CardDisplayMode {
	m := CardDisplayMode(s)
	return &m
}

func cardDisplayContext(s string) *CardDisplayContext {
	c := CardDisplayContext(s)
	return &c
}

// --- optionalString ---

func TestOptionalString_Nil(t *testing.T) {
	t.Parallel()
	result := optionalString(nil)
	if !result.IsNull() {
		t.Errorf("expected null, got %v", result)
	}
}

func TestOptionalString_NonNil(t *testing.T) {
	t.Parallel()
	result := optionalString(strPtr("hello"))
	if result.IsNull() || result.ValueString() != "hello" {
		t.Errorf("expected \"hello\", got %v", result)
	}
}

// --- mapOptionsFromAPI ---

func TestMapOptionsFromAPI_Nil(t *testing.T) {
	t.Parallel()
	if mapOptionsFromAPI(nil) != nil {
		t.Error("expected nil for nil input")
	}
}

func TestMapOptionsFromAPI_BothFieldsPresent(t *testing.T) {
	t.Parallel()
	opts := &GameOptions{
		CardDisplayMode:    cardDisplayMode("managed"),
		CardDisplayContext: cardDisplayContext("everywhere"),
	}
	result := mapOptionsFromAPI(opts)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.CardDisplayMode.ValueString() != "managed" {
		t.Errorf("expected CardDisplayMode=managed, got %s", result.CardDisplayMode.ValueString())
	}
	if result.CardDisplayContext.ValueString() != "everywhere" {
		t.Errorf("expected CardDisplayContext=everywhere, got %s", result.CardDisplayContext.ValueString())
	}
}

func TestMapOptionsFromAPI_NilFields(t *testing.T) {
	t.Parallel()
	result := mapOptionsFromAPI(&GameOptions{})
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.CardDisplayMode.IsNull() {
		t.Error("expected CardDisplayMode to be null")
	}
	if !result.CardDisplayContext.IsNull() {
		t.Error("expected CardDisplayContext to be null")
	}
}

// --- mapGridFromAPI ---

func TestMapGridFromAPI_Nil(t *testing.T) {
	t.Parallel()
	if mapGamePlayDataFromAPI(nil) != nil {
		t.Error("expected nil for nil input")
	}
}

func TestMapGridFromAPI_EmptySlots(t *testing.T) {
	t.Parallel()
	result := mapGamePlayDataFromAPI(&GamePlayData{PlayerCount: 2, Slots: nil})
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.PlayerCount.ValueInt64() != 2 {
		t.Errorf("expected PlayerCount=2, got %d", result.PlayerCount.ValueInt64())
	}
	if len(result.Slots) != 0 {
		t.Errorf("expected 0 slots, got %d", len(result.Slots))
	}
}

func TestMapGridFromAPI_SlotWithPlayerOwner(t *testing.T) {
	t.Parallel()
	data := &GamePlayData{
		PlayerCount: 1,
		Slots: []GridSlot{
			{
				Row: 0, Column: 0, Width: 2, Height: 1,
				Type:        SlotType("cards"),
				MaxCount:    5,
				Visibility:  SlotVisibility("public"),
				PlayerOwner: intPtr(1),
			},
		},
	}
	result := mapGamePlayDataFromAPI(data)
	if len(result.Slots) != 1 {
		t.Fatalf("expected 1 slot, got %d", len(result.Slots))
	}
	slot := result.Slots[0]
	if slot.PlayerOwner.IsNull() || slot.PlayerOwner.ValueInt64() != 1 {
		t.Errorf("expected PlayerOwner=1, got %v", slot.PlayerOwner)
	}
	if slot.Type.ValueString() != "cards" {
		t.Errorf("expected Type=cards, got %s", slot.Type.ValueString())
	}
	if slot.Visibility.ValueString() != "public" {
		t.Errorf("expected Visibility=public, got %s", slot.Visibility.ValueString())
	}
}

func TestMapGridFromAPI_SlotWithoutPlayerOwner(t *testing.T) {
	t.Parallel()
	data := &GamePlayData{
		PlayerCount: 1,
		Slots: []GridSlot{
			{
				Row: 1, Column: 2, Width: 1, Height: 1,
				Type:       SlotType("counters"),
				MaxCount:   10,
				Visibility: SlotVisibility("private"),
			},
		},
	}
	result := mapGamePlayDataFromAPI(data)
	slot := result.Slots[0]
	if !slot.PlayerOwner.IsNull() {
		t.Errorf("expected PlayerOwner to be null, got %v", slot.PlayerOwner)
	}
}

// --- newGameOptions ---

func TestNewGameOptions_NilModel(t *testing.T) {
	t.Parallel()
	// newGameOptions does not accept nil — it expects a valid model pointer.
	// Callers guard with nil check, so test with an empty model instead.
	result := newGameOptions(&gameOptionsModel{
		CardDisplayMode:    types.StringNull(),
		CardDisplayContext: types.StringNull(),
	})
	if result == nil {
		t.Fatal("expected non-nil GameOptions")
	}
	if result.CardDisplayMode != nil {
		t.Error("expected CardDisplayMode to be nil for null input")
	}
	if result.CardDisplayContext != nil {
		t.Error("expected CardDisplayContext to be nil for null input")
	}
}

func TestNewGameOptions_FullModel(t *testing.T) {
	t.Parallel()
	result := newGameOptions(&gameOptionsModel{
		CardDisplayMode:    types.StringValue("imageonly"),
		CardDisplayContext: types.StringValue("ingameonly"),
	})
	if result.CardDisplayMode == nil || string(*result.CardDisplayMode) != "imageonly" {
		t.Errorf("expected CardDisplayMode=imageonly, got %v", result.CardDisplayMode)
	}
	if result.CardDisplayContext == nil || string(*result.CardDisplayContext) != "ingameonly" {
		t.Errorf("expected CardDisplayContext=ingameonly, got %v", result.CardDisplayContext)
	}
}

// --- newGamePlayData ---

func TestNewGamePlayData_EmptySlots(t *testing.T) {
	t.Parallel()
	result := newGamePlayData(&gamePlayDataModel{
		PlayerCount: types.Int64Value(4),
		Slots:       nil,
	})
	if result.PlayerCount != 4 {
		t.Errorf("expected PlayerCount=4, got %d", result.PlayerCount)
	}
	if len(result.Slots) != 0 {
		t.Errorf("expected 0 slots, got %d", len(result.Slots))
	}
}

func TestNewGamePlayData_SlotWithPlayerOwner(t *testing.T) {
	t.Parallel()
	model := &gamePlayDataModel{
		PlayerCount: types.Int64Value(2),
		Slots: []gameSlotModel{
			{
				Row:         types.Int64Value(0),
				Column:      types.Int64Value(1),
				Width:       types.Int64Value(3),
				Height:      types.Int64Value(2),
				Type:        types.StringValue("cards"),
				MaxCount:    types.Int64Value(7),
				Visibility:  types.StringValue("private"),
				PlayerOwner: types.Int64Value(2),
			},
		},
	}
	result := newGamePlayData(model)
	if len(result.Slots) != 1 {
		t.Fatalf("expected 1 slot, got %d", len(result.Slots))
	}
	slot := result.Slots[0]
	if slot.PlayerOwner == nil || *slot.PlayerOwner != 2 {
		t.Errorf("expected PlayerOwner=2, got %v", slot.PlayerOwner)
	}
	if slot.Row != 0 || slot.Column != 1 || slot.Width != 3 || slot.Height != 2 {
		t.Errorf("unexpected slot dimensions: %+v", slot)
	}
}

func TestNewGamePlayData_SlotWithoutPlayerOwner(t *testing.T) {
	t.Parallel()
	model := &gamePlayDataModel{
		PlayerCount: types.Int64Value(1),
		Slots: []gameSlotModel{
			{
				Row:         types.Int64Value(0),
				Column:      types.Int64Value(0),
				Width:       types.Int64Value(1),
				Height:      types.Int64Value(1),
				Type:        types.StringValue("counters"),
				MaxCount:    types.Int64Value(3),
				Visibility:  types.StringValue("public"),
				PlayerOwner: types.Int64Null(),
			},
		},
	}
	result := newGamePlayData(model)
	slot := result.Slots[0]
	if slot.PlayerOwner != nil {
		t.Errorf("expected PlayerOwner to be nil, got %v", slot.PlayerOwner)
	}
}

// --- round-trip: mapGridFromAPI <-> newGamePlayData ---

func TestGridRoundTrip(t *testing.T) {
	t.Parallel()
	original := &GamePlayData{
		PlayerCount: 3,
		Slots: []GridSlot{
			{Row: 0, Column: 0, Width: 2, Height: 1, Type: "cards", MaxCount: 4, Visibility: "public", PlayerOwner: intPtr(1)},
			{Row: 1, Column: 2, Width: 1, Height: 2, Type: "counters", MaxCount: 8, Visibility: "private"},
		},
	}
	model := mapGamePlayDataFromAPI(original)
	result := newGamePlayData(model)

	if result.PlayerCount != original.PlayerCount {
		t.Errorf("PlayerCount mismatch: want %d, got %d", original.PlayerCount, result.PlayerCount)
	}
	for i, want := range original.Slots {
		got := result.Slots[i]
		if got.Row != want.Row || got.Column != want.Column || got.Width != want.Width || got.Height != want.Height {
			t.Errorf("slot[%d] dimensions mismatch: want %+v, got %+v", i, want, got)
		}
		if string(got.Type) != string(want.Type) {
			t.Errorf("slot[%d] Type mismatch: want %s, got %s", i, want.Type, got.Type)
		}
		if string(got.Visibility) != string(want.Visibility) {
			t.Errorf("slot[%d] Visibility mismatch: want %s, got %s", i, want.Visibility, got.Visibility)
		}
		switch {
		case want.PlayerOwner == nil && got.PlayerOwner != nil:
			t.Errorf("slot[%d] expected nil PlayerOwner, got %d", i, *got.PlayerOwner)
		case want.PlayerOwner != nil && (got.PlayerOwner == nil || *got.PlayerOwner != *want.PlayerOwner):
			t.Errorf("slot[%d] PlayerOwner mismatch: want %d, got %v", i, *want.PlayerOwner, got.PlayerOwner)
		}
	}
}
