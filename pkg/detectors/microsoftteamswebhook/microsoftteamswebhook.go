package microsoftteamswebhook

import (
	"context"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/trufflesecurity/trufflehog/v3/pkg/common"
	"github.com/trufflesecurity/trufflehog/v3/pkg/detectors"
	"github.com/trufflesecurity/trufflehog/v3/pkg/pb/detectorspb"
)

type Scanner struct{}

// Ensure the Scanner satisfies the interface at compile time
var _ detectors.Detector = (*Scanner)(nil)

var (
	client = common.SaneHttpClient()

	//Make sure that your group is surrounded in boundry characters such as below to reduce false positives
	keyPat = regexp.MustCompile(`(https:\/\/[a-zA-Z-0-9]+\.webhook\.office\.com\/webhookb2\/[a-zA-Z-0-9]{8}-[a-zA-Z-0-9]{4}-[a-zA-Z-0-9]{4}-[a-zA-Z-0-9]{4}-[a-zA-Z-0-9]{12}\@[a-zA-Z-0-9]{8}-[a-zA-Z-0-9]{4}-[a-zA-Z-0-9]{4}-[a-zA-Z-0-9]{4}-[a-zA-Z-0-9]{12}\/IncomingWebhook\/[a-zA-Z-0-9]{32}\/[a-zA-Z-0-9]{8}-[a-zA-Z-0-9]{4}-[a-zA-Z-0-9]{4}-[a-zA-Z-0-9]{4}-[a-zA-Z-0-9]{12})`)
)

// Keywords are used for efficiently pre-filtering chunks.
// Use identifiers in the secret preferably, or the provider name.
func (s Scanner) Keywords() []string {
	return []string{"microsoft"}
}

// FromData will find and optionally verify MicrosoftTeamsWebhook secrets in a given set of bytes.
func (s Scanner) FromData(ctx context.Context, verify bool, data []byte) (results []detectors.Result, err error) {
	dataStr := string(data)

	matches := keyPat.FindAllStringSubmatch(dataStr, -1)

	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		resMatch := strings.TrimSpace(match[1])

		s1 := detectors.Result{
			DetectorType: detectorspb.DetectorType_MicrosoftTeamsWebhook,
			Raw:          []byte(resMatch),
		}
		if verify {
			payload := strings.NewReader(`{'text':'This is a verification message from TruffleHog. It means that there has been a live webhook credential found.'}`)
			req, err := http.NewRequestWithContext(ctx, "POST", resMatch, payload)
			if err != nil {
				continue
			}
			req.Header.Add("Content-Type", "application/json")
			res, err := client.Do(req)
			if err != nil {
				continue
			}
			defer res.Body.Close()

			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode >= 200 && string(body) == "1" {
				s1.Verified = true
			} else {
				//This function will check false positives for common test words, but also it will make sure the key appears 'random' enough to be a real key
				if detectors.IsKnownFalsePositive(resMatch, detectors.DefaultFalsePositives, false) {
					continue
				}
			}
		}

		results = append(results, s1)
	}

	return detectors.CleanResults(results), nil
}
