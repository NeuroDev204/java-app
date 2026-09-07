package vm

import "testing"

func TestAdjustKnightOnlinePotionFallback(t *testing.T) {
	tests := []struct {
		name         string
		callerClass  string
		callerMethod string
		targetClass  string
		targetMethod string
		descriptor   string
		input        int32
		want         int32
	}{
		{
			name:         "first potion uses first potion image",
			callerClass:  "bk",
			callerMethod: "e",
			targetClass:  "bk",
			targetMethod: "o",
			descriptor:   "(S)Laq;",
			input:        4000,
			want:         4001,
		},
		{
			name:         "later potion keeps one based image numbering",
			callerClass:  "bk",
			callerMethod: "e",
			targetClass:  "bk",
			targetMethod: "o",
			descriptor:   "(S)Laq;",
			input:        4077,
			want:         4078,
		},
		{
			name:         "item fallback is unchanged",
			callerClass:  "bk",
			callerMethod: "d",
			targetClass:  "bk",
			targetMethod: "o",
			descriptor:   "(S)Laq;",
			input:        2000,
			want:         2000,
		},
		{
			name:         "unrelated caller is unchanged",
			callerClass:  "other",
			callerMethod: "e",
			targetClass:  "bk",
			targetMethod: "o",
			descriptor:   "(S)Laq;",
			input:        4000,
			want:         4000,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parameters := []Slot{{Num: test.input}}
			adjustKnightOnlinePotionFallback(
				test.callerClass,
				test.callerMethod,
				test.targetClass,
				test.targetMethod,
				test.descriptor,
				parameters,
			)
			if parameters[0].Num != test.want {
				t.Fatalf("fallback image id = %d, want %d", parameters[0].Num, test.want)
			}
		})
	}
}
