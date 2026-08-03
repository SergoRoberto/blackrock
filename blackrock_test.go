package blackrock

import (
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	type args struct {
		rangez int64
		seed   int64
	}
	tests := []struct {
		name string
		args func(t *testing.T) args

		want1 *BlackRock
	}{
		{
			name: "must solve the square root and increment B while the result is less than rangez",
			args: func(*testing.T) args {
				return args{
					rangez: 4,
					seed:   1,
				}
			},
			want1: &BlackRock{
				Rounds: 3,
				Seed:   1,
				Range:  4,
				A:      1,
				B:      5,
			},
		},
		{
			name: "if split is zero the value of A must be 1",
			args: func(*testing.T) args {
				return args{
					rangez: 0,
					seed:   1,
				}
			},
			want1: &BlackRock{
				Rounds: 3,
				Seed:   1,
				Range:  0,
				A:      1,
				B:      1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tArgs := tt.args(t)

			got1 := New(tArgs.rangez, tArgs.seed)

			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("NewBlackRock got1 = %v, want1: %v", got1, tt.want1)
			}
		})
	}
}

var testSeeds = []int64{0, 1, 42, 12345}

// UnShuffle must invert Shuffle over the whole domain. This is the property the
// caller relies on to map a shuffled index back to the original one.
func TestShuffleUnShuffleRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		rangez  int64
		samples int64 // 0 means walk the whole range
	}{
		{name: "tiny range", rangez: 1},
		{name: "range smaller than the split", rangez: 4},
		{name: "small range", rangez: 1000},
		{name: "medium range", rangez: 100000, samples: 20000},
		{name: "ipv4 sized range", rangez: 651399200, samples: 20000},
		{name: "range wider than 32 bits", rangez: 6300000000, samples: 20000},
	}

	for _, tt := range tests {
		for _, seed := range testSeeds {
			t.Run(tt.name, func(t *testing.T) {
				blackrock := New(tt.rangez, seed)

				step := int64(1)
				if tt.samples > 0 && tt.rangez > tt.samples {
					step = tt.rangez / tt.samples
				}

				for i := int64(0); i < tt.rangez; i += step {
					shuffled := blackrock.Shuffle(i)
					if got := blackrock.UnShuffle(shuffled); got != i {
						t.Fatalf("UnShuffle(Shuffle(%d)) = %d, want %d (range %d, seed %d, shuffled %d)",
							i, got, i, tt.rangez, seed, shuffled)
					}
				}
			})
		}
	}
}

// Shuffle must be a permutation of [0, Range): every input maps to a distinct
// output that stays inside the range.
func TestShuffleIsPermutation(t *testing.T) {
	for _, rangez := range []int64{1, 4, 1000, 65536} {
		for _, seed := range testSeeds {
			blackrock := New(rangez, seed)
			seen := make([]bool, rangez)

			for i := int64(0); i < rangez; i++ {
				shuffled := blackrock.Shuffle(i)

				if shuffled < 0 || shuffled >= rangez {
					t.Fatalf("Shuffle(%d) = %d, out of range [0, %d) (seed %d)", i, shuffled, rangez, seed)
				}
				if seen[shuffled] {
					t.Fatalf("Shuffle(%d) = %d, already produced by an earlier input (range %d, seed %d)",
						i, shuffled, rangez, seed)
				}

				seen[shuffled] = true
			}
		}
	}
}

// Unfe must invert Fe for both an odd and an even number of rounds. Each parity
// exercises a different branch of the reverse construction, so an error in one
// of them stays invisible when only the default round count is tested.
func TestUnfeInvertsFe(t *testing.T) {
	blackrock := New(100000, 7)

	for _, rounds := range []int64{1, 2, 3} {
		for _, seed := range testSeeds {
			for m := int64(0); m < 5000; m++ {
				encrypted := blackrock.Fe(rounds, blackrock.A, blackrock.B, m, seed)
				if got := blackrock.Unfe(rounds, blackrock.A, blackrock.B, encrypted, seed); got != m {
					t.Fatalf("Unfe(Fe(%d)) = %d, want %d (rounds %d, seed %d)", m, got, m, rounds, seed)
				}
			}
		}
	}
}

// The reverse construction subtracts R from the round function output before
// wrapping it. Getting that sign wrong only shows up when F(j, L, seed) exceeds
// R, which is the common case but was never covered by a test.
func TestUnfeSubtractsRoundOutput(t *testing.T) {
	const (
		rounds = 3
		seed   = 1
	)

	blackrock := New(1000, seed)
	exercised := false

	for m := int64(0); m < 1000; m++ {
		L := m % blackrock.A
		R := m / blackrock.A
		if blackrock.F(rounds, L, seed) > R {
			exercised = true
		}

		encrypted := blackrock.Fe(rounds, blackrock.A, blackrock.B, m, seed)
		if got := blackrock.Unfe(rounds, blackrock.A, blackrock.B, encrypted, seed); got != m {
			t.Fatalf("Unfe(Fe(%d)) = %d, want %d", m, got, m)
		}
	}

	if !exercised {
		t.Fatal("no input reached the F(j, L, seed) > R branch, the test no longer covers it")
	}
}
