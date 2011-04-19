// Twackup -- Backs up your tweets
//
//
// Only works on non-protected accounts. Why would you be on twitter
// if you're not being public?


// http://api.twitter.com/version/statuses/user_timeline.format


package main

import (
	"fmt"
	"http"
	"io/ioutil"
	"json"
	"log"
	"os"
	"strconv"
	"strings"
)

func FindEndpoints(dir string) (oldest, newest uint64, err os.Error) {
	file, err := os.Open(dir)
	if err != nil {
		return
	}
	defer file.Close()
	for {
		list, err := file.Readdirnames(1000)
		if err != nil {
			return
		}
		if len(list) == 0 {
			break
		}
		for _, name := range list {
			JSON_SUFFIX := ".json"
			if strings.HasSuffix(name, JSON_SUFFIX) {
				name = name[:len(name)-len(JSON_SUFFIX)]
				seen, err := strconv.Atoui64(name)
				if err != nil {
					// it's an unrelated json file?
					continue
				}
				if oldest == 0 || seen < oldest {
					oldest = seen
				}
				if newest == 0 || seen > newest {
					newest = seen
				}
			}
		}
	}
	return
}

// Gets tweets from max_id backwards.
// max_id==0 means latest tweet.
func GetTweets(user string, max_id uint64, since_id uint64) (tweets []map[string]interface{}, err os.Error) {
	args := map[string][]string{
		"screen_name":      []string{user},
		"trim_user":        []string{"true"},
		"include_rts":      []string{"true"},
		"include_entities": []string{"true"},
		"count":            []string{"200"},
	}
	if max_id != 0 {
		args["max_id"] = []string{strconv.Uitoa64(max_id)}
	}
	if since_id != 0 {
		args["since_id"] = []string{strconv.Uitoa64(since_id)}
	}
	query := http.EncodeQuery(args)
	url := "http://api.twitter.com/1/statuses/user_timeline.json?" + query
	log.Printf("Fetching url %v\n", url)
	r, _, err := http.Get(url)
	if err != nil {
		return
	}
	ctype := r.Header.Get("Content-Type")
	if ctype != "application/json; charset=utf-8" {
		return
	}

	//TODO ponder rate-limiting http headers
	//X-Ratelimit-Remaining: 143
	//X-Ratelimit-Limit: 150
	//X-Ratelimit-Reset: 1303182615

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}
	err = r.Body.Close()
	if err != nil {
		return
	}
	err = json.Unmarshal(buf, &tweets)
	return
}

func IdFromTweet(tweet map[string]interface{}) (id uint64, err os.Error) {
	id_raw := tweet["id_str"]
	id_str, ok := id_raw.(string)
	if !ok {
		msg := fmt.Sprintf("tweet id is not a string: %v", id_raw)
		err = os.NewError(msg)
		return
	}

	id, err = strconv.Atoui64(id_str)
	if err != nil {
		return
	}
	return
}

func SaveTweet(dir string, tweet map[string]interface{}) (id uint64, err os.Error) {
	// clean up Twitter's mistakes
	tweet["id"] = nil, false

	id, err = IdFromTweet(tweet)
	if err != nil {
		return
	}
	out, err := json.MarshalIndent(tweet, "", "  ")
	if err != nil {
		return
	}

	// roundtrip it from number back to string to canonicalize it
	filename := dir + "/" + strconv.Uitoa64(id) + ".json"
	tmp := filename + "." + strconv.Itoa(os.Getpid()) + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return
	}
	_, err = f.Write(out)
	if err != nil {
		_ = os.Remove(tmp)
		return
	}
	err = f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return
	}

	err = os.Rename(tmp, filename)
	if err != nil {
		_ = os.Remove(tmp)
		return
	}
	return
}

// Get tweets backwards in history until end of time (or Twitter API
// limit).
func getOldTweets(dir string, user string, oldest uint64) (err os.Error) {
	for {
		var max_id uint64
		if oldest != 0 {
			// max_id is inclusive, so do -1 to avoid duplicating
			// the one we have
			max_id = oldest - 1
		}
		tweets, err := GetTweets(user, max_id, 0)
		if err != nil {
			return
		}
		if len(tweets) == 0 {
			break
		}
		for _, tweet := range tweets {
			_, err := SaveTweet(dir, tweet)
			if err != nil {
				return
			}
		}
		oldest, err = IdFromTweet(tweets[len(tweets)-1])
		if err != nil {
			return
		}
		log.Printf("Saved %d old tweets\n", len(tweets))
	}
	return
}

// Get tweets forwards in time; since Twitter gives you the *latest*
// chunk, not the oldest chunk, we need to save these to disk in
// reverse order.
func getNewTweets(dir string, user string, newest uint64) (err os.Error) {
	// gather them in RAM
	var tweets []map[string]interface{}
	for {
		var since_id uint64
		if newest != 0 {
			since_id = newest
		}
		chunk, err := GetTweets(user, 0, since_id)
		if err != nil {
			return
		}
		if len(chunk) == 0 {
			break
		}
		tweets = append(tweets, chunk...)

		newest, err = IdFromTweet(tweets[0])
		if err != nil {
			return
		}
	}

	// now save them to disk in reverse order
	for i := len(tweets) - 1; i >= 0; i-- {
		tweet := tweets[i]
		_, err := SaveTweet(dir, tweet)
		if err != nil {
			return
		}
	}
	log.Printf("Saved %d new tweets\n", len(tweets))
	return
}

func main() {
	log.SetFlags(0)
	if len(os.Args) != 3 || os.Args[1][0] == '-' {
		log.Printf("%s: usage: %s USER DIR", os.Args[0], os.Args[0])
		os.Exit(2)
	}
	user := os.Args[1]
	dir := os.Args[2]

	oldest, newest, err := FindEndpoints(dir)
	if err != nil {
		log.Fatalf("%s: %s", os.Args[0], err)
	}

	// usually prioritize fetching new tweets first, but not on
	// the first run; it'd load them all to RAM, where as
	// getOldTweets is more efficient
	if newest != 0 {
		log.Printf("Fetching tweets newer than %d\n", newest)
		getNewTweets(dir, user, newest)
	}

	log.Printf("Fetching tweets older than %d\n", oldest)
	getOldTweets(dir, user, oldest)
}
