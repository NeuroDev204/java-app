package vm

import "testing"

func TestResLocalIconIDMatchesKnightOnlineResourceFallbacks(t *testing.T) {
	tests := []struct {
		path string
		want int32
	}{
		{path: "/x1/resLocal/npc/123.png", want: 3123},
		{path: "/x1/resLocal/npc/icon/327.png", want: 3827},
		{path: "/x1/resLocal/tree/7.png", want: 7},
		{path: "/x1/resLocal/mons/42.png", want: 1042},
		{path: "/x1/resLocal/icon/icon_1.png", want: 2001},
		{path: "/x1/resLocal/iconq/1.png", want: 5001},
		{path: "/x1/resLocal/potion/potionIcon_03.png", want: 4002},
		{path: "/x1/resLocal/material/0.png", want: 5500},
		{path: "/x1/resLocal/skill/iconSkill_03.png", want: 6003},
		{path: "/x1/resLocal/iconclan/12.png", want: 7012},
		{path: "/x1/resLocal/iconclan/shop/327.png", want: 7827},
		{path: "/x1/resLocal/iconclan/huyhieu/4.png", want: 9504},
		{path: "/x1/resLocal/pet/icon/icon_pet_0.png", want: 10000},
		{path: "/x1/resLocal/pet/22.png", want: 10222},
	}

	for _, test := range tests {
		got, ok := resLocalIconID(test.path)
		if !ok {
			t.Errorf("resLocalIconID(%q) did not recognize a valid resource", test.path)
			continue
		}
		if got != test.want {
			t.Errorf("resLocalIconID(%q) = %d, want %d", test.path, got, test.want)
		}
	}
}
