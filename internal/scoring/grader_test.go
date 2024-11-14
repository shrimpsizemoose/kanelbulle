package scoring

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestGrader_CalculateScore(t *testing.T) {
    testCases := []struct {
        name          string
        baseScore     int
        deadline      int64
        submitTime    int64
        expectedScore int
    }{
        {
            name:          "On-time submission",
            baseScore:     10,
            deadline:      1680288000, // April 1, 2023
            submitTime:    1680288000,
            expectedScore: 10,
        },
        {
            name:          "Late submission with modifier",
            baseScore:     10,
            deadline:      1680288000, // April 1, 2023
            submitTime:    1680374400, // April 2, 2023
            expectedScore: 9,
        },
        {
            name:          "Late submission with default penalty",
            baseScore:     10,
            deadline:      1680288000, // April 1, 2023
            submitTime:    1680547200, // April 4, 2023
            expectedScore: 7,
        },
        {
            name:          "Late submission with extra penalty",
            baseScore:     10,
            deadline:      1680288000, // April 1, 2023
            submitTime:    1680806400, // April 8, 2023
            expectedScore: 0,
        },
        {
            name:          "Negative score capped at 0",
            baseScore:     10,
            deadline:      1680288000, // April 1, 2023
            submitTime:    1681411200, // April 15, 2023
            expectedScore: 0,
        },
    }

    grader := &Grader{
        LateDaysModifiers: map[int]int{
            1: -1,
            2: -2,
            3: -3,
        },
        DefaultLatePenalty: 0.7,
        MaxLateDays:        5,
        ExtraLatePenalty:   5,
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            score := grader.CalculateScore(tc.baseScore, tc.deadline, tc.submitTime)
            assert.Equal(t, tc.expectedScore, score)
        })
    }
}
