package manageai

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	_ "embed"
)

//go:embed redirections.json
var redirectionsJSON []byte

var redirectHelper *ModelRedirectHelper
var redirectHelperOnce sync.Once

type Redirection struct {
	ModelId string `json:"modelId"`
	// some model is deprecated, so we need to redirect to another model
	RedirectTo *string `json:"redirectTo"`
	// some model is not support multi tools, so we need to redirect to another model
	IfMultiToolsRedirectTo *string `json:"ifMultiToolsRedirectTo"`
}

type AliasModelId struct {
	Alias   string `json:"alias"`
	ModelId string `json:"modelId"`
}

type ModelRedirectHelper struct {
	AliasModelIds []AliasModelId `json:"aliasModelIds"`
	Redirections  []Redirection  `json:"redirections"`
}

func levenshteinDistance(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	// Use two rows instead of a full matrix for O(min(la,lb)) space
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(
				prev[j]+1,      // deletion
				curr[j-1]+1,    // insertion
				prev[j-1]+cost, // substitution
			)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min(vals ...int) int {
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func GetRedirectHelper() *ModelRedirectHelper {
	redirectHelperOnce.Do(func() {
		redirectHelper = &ModelRedirectHelper{}
		if err := json.Unmarshal(redirectionsJSON, &redirectHelper); err != nil {
			panic(err)
		}
	})
	return redirectHelper
}

func (rh *ModelRedirectHelper) GetModelId(modelId *string, MultiTools bool) (*string, error) {
	if modelId == nil {
		return nil, nil // model id allow to be nil, which mean using default model id
	}
	normalizedModelId := strings.ToLower(*modelId)
	if len(rh.AliasModelIds) > 0 {
		for _, aliasModelId := range rh.AliasModelIds {
			if aliasModelId.Alias == normalizedModelId {
				normalizedModelId = aliasModelId.ModelId
			}
		}
	}

	if len(rh.Redirections) > 0 {
		for _, redirection := range rh.Redirections {
			if redirection.ModelId == normalizedModelId {
				if MultiTools {
					if redirection.IfMultiToolsRedirectTo != nil {
						return redirection.IfMultiToolsRedirectTo, nil
					} else if redirection.RedirectTo != nil {
						return redirection.RedirectTo, nil
					} else {
						return &normalizedModelId, nil
					}
				} else {
					if redirection.RedirectTo != nil {
						return redirection.RedirectTo, nil
					} else {
						return &normalizedModelId, nil
					}
				}
			}
		}
	}

	closest, dist := rh.findClosestModel(normalizedModelId)
	if dist <= 5 { // reasonable threshold
		return nil, fmt.Errorf("cannot recognize model id %q, did you mean %q?", *modelId, closest)
	}

	return nil, fmt.Errorf("cannot recognize model id %q", *modelId)
}

func (rh *ModelRedirectHelper) findClosestModel(input string) (string, int) {
	bestMatch := ""
	bestDist := len(input) + 1
	// Check aliases
	for _, a := range rh.AliasModelIds {
		d := levenshteinDistance(input, a.Alias)
		if d < bestDist {
			bestDist = d
			bestMatch = a.Alias
		}
	}
	// Check redirections
	for _, r := range rh.Redirections {
		d := levenshteinDistance(input, r.ModelId)
		if d < bestDist {
			bestDist = d
			bestMatch = r.ModelId
		}
	}
	return bestMatch, bestDist
}
