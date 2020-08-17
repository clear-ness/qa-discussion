package model

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	TERMS_TYPE_PLAIN   = "plain_terms"
	TERMS_TYPE_TITLE   = "title_terms"
	TERMS_TYPE_TAG     = "tag_terms"
	TERMS_TYPE_BODY    = "body_terms"
	TERMS_TYPE_SIMILAR = "similar_terms"
	TERMS_TYPE_LINK    = "link_terms"
)

type SearchParams struct {
	Terms          string
	ExcludedTerms  string
	TermsType      string
	PostType       string
	User           string
	Parent         string
	Ids            []string
	MinVotes       *int
	MaxVotes       *int
	MinAnswers     *int
	MaxAnswers     *int
	FromDate       string
	ToDate         string
	TimeZoneOffset int
}

var searchTermPuncStart = regexp.MustCompile(`^[^\pL\d\s#"]+`)
var searchTermPuncEnd = regexp.MustCompile(`[^\pL\d\s*"]+$`)
var dupTagStart = regexp.MustCompile(`^#{2,}`)

var searchFlags = [...]string{"is", "user", "minvotes", "maxvotes", "body", "from", "to", "minanswers", "maxanswers", "title", "inquestion"}

func (p *SearchParams) GetFromDateMillis() int64 {
	date, err := time.Parse("2006-01-02", PadDateStringZeros(p.FromDate))
	if err != nil {
		return 0
	}
	return GetStartOfDayMillis(date, p.TimeZoneOffset)
}

func (p *SearchParams) GetToDateMillis() int64 {
	date, err := time.Parse("2006-01-02", PadDateStringZeros(p.ToDate))
	if err != nil {
		date = time.Now()
	}
	return GetEndOfDayMillis(date, p.TimeZoneOffset)
}

type flag struct {
	name    string
	value   string
	exclude bool
}

type searchWord struct {
	value   string
	exclude bool
}

func splitWords(text string) []string {
	words := []string{}

	foundQuote := false
	location := 0
	for i, char := range text {
		if char == '"' {
			if foundQuote {
				// Grab the quoted section
				word := text[location : i+1]
				words = append(words, word)
				foundQuote = false
				location = i + 1
			} else {
				nextStart := i
				if i > 0 && text[i-1] == '-' {
					nextStart = i - 1
				}
				words = append(words, strings.Fields(text[location:nextStart])...)
				foundQuote = true
				location = nextStart
			}
		}
	}

	words = append(words, strings.Fields(text[location:])...)

	return words
}

func parseSearchFlags(input []string) ([]searchWord, []flag) {
	words := []searchWord{}
	flags := []flag{}

	skipNextWord := false
	for i, word := range input {
		if skipNextWord {
			skipNextWord = false
			continue
		}

		isFlag := false

		if colon := strings.Index(word, ":"); colon != -1 {
			var flagName string
			var exclude bool
			if strings.HasPrefix(word, "-") {
				flagName = word[1:colon]
				exclude = true
			} else {
				flagName = word[:colon]
				exclude = false
			}

			value := word[colon+1:]

			for _, searchFlag := range searchFlags {
				if strings.EqualFold(flagName, searchFlag) {
					if value != "" {
						flags = append(flags, flag{
							searchFlag,
							value,
							exclude,
						})
						isFlag = true
					} else if i < len(input)-1 {
						flags = append(flags, flag{
							searchFlag,
							input[i+1],
							exclude,
						})
						skipNextWord = true
						isFlag = true
					}

					if isFlag {
						break
					}
				}
			}
		}

		// plainTerm or tag
		if !isFlag {
			exclude := false
			if strings.HasPrefix(word, "-") {
				exclude = true
			}
			// trim off surrounding punctuation (note that we leave trailing asterisks to allow wildcards)
			word = searchTermPuncStart.ReplaceAllString(word, "")
			word = searchTermPuncEnd.ReplaceAllString(word, "")

			// and remove extra pound #s
			word = dupTagStart.ReplaceAllString(word, "#")

			if len(word) != 0 {
				words = append(words, searchWord{
					word,
					exclude,
				})
			}
		}
	}

	return words, flags
}

func ParseSearchParams(text string, timeZoneOffset int) []*SearchParams {
	words, flags := parseSearchFlags(splitWords(text))

	tagTermList := []string{}
	excludedTagTermList := []string{}
	plainTermList := []string{}
	excludedPlainTermList := []string{}

	for _, word := range words {
		if validSharpTag.MatchString(word.value) {
			word.value = tagStart.ReplaceAllString(word.value, "")

			if word.exclude {
				excludedTagTermList = append(excludedTagTermList, word.value)
			} else {
				tagTermList = append(tagTermList, word.value)
			}
		} else {
			if word.exclude {
				excludedPlainTermList = append(excludedPlainTermList, word.value)
			} else {
				plainTermList = append(plainTermList, word.value)
			}
		}
	}

	tagTerms := strings.Join(tagTermList, " ")
	excludedTagTerms := strings.Join(excludedTagTermList, " ")
	plainTerms := strings.Join(plainTermList, " ")
	excludedPlainTerms := strings.Join(excludedPlainTermList, " ")

	postType := ""
	user := ""
	var minVotes *int
	var maxVotes *int
	fromDate := ""
	toDate := ""
	var minAnswers *int
	var maxAnswers *int
	parent := ""

	bodyTermList := []string{}
	excludedBodyTermList := []string{}
	titleTermList := []string{}
	excludedTitleTermList := []string{}

	for _, flag := range flags {
		if flag.name == "is" {
			if IsQuestionOrAnswer(flag.value) {
				postType = flag.value
			}
		} else if flag.name == "user" {
			if len(flag.value) == 26 {
				user = flag.value
			}
		} else if flag.name == "minvotes" {
			if val, err := strconv.Atoi(flag.value); err == nil {
				minVotes = &val
			}
		} else if flag.name == "maxvotes" {
			if val, err := strconv.Atoi(flag.value); err == nil {
				maxVotes = &val
			}
		} else if flag.name == "body" {
			if flag.exclude {
				excludedBodyTermList = append(excludedBodyTermList, flag.value)
			} else {
				bodyTermList = append(bodyTermList, flag.value)
			}
		} else if flag.name == "from" {
			fromDate = flag.value
		} else if flag.name == "to" {
			toDate = flag.value
		} else if flag.name == "minanswers" {
			if val, err := strconv.Atoi(flag.value); err == nil {
				minAnswers = &val
			}
		} else if flag.name == "maxanswers" {
			if val, err := strconv.Atoi(flag.value); err == nil {
				maxAnswers = &val
			}
		} else if flag.name == "title" {
			if flag.exclude {
				excludedTitleTermList = append(excludedTitleTermList, flag.value)
			} else {
				titleTermList = append(titleTermList, flag.value)
			}
		} else if flag.name == "inquestion" {
			if len(flag.value) == 26 {
				parent = flag.value
			}
		}
	}

	bodyTerms := strings.Join(bodyTermList, " ")
	excludedBodyTerms := strings.Join(excludedBodyTermList, " ")
	titleTerms := strings.Join(titleTermList, " ")
	excludedTitleTerms := strings.Join(excludedTitleTermList, " ")

	paramsList := []*SearchParams{}

	if len(tagTerms) > 0 || len(excludedTagTerms) > 0 ||
		len(titleTerms) > 0 || len(excludedTitleTerms) > 0 ||
		minAnswers != nil || maxAnswers != nil {
		postType = POST_TYPE_QUESTION
	}

	// special case
	// e.g. is:question and inquestion:question_id
	var ids []string
	if postType == POST_TYPE_QUESTION && parent != "" {
		ids = append(ids, parent)
	}

	if len(plainTerms) > 0 || len(excludedPlainTerms) > 0 {
		paramsList = append(paramsList, &SearchParams{
			Terms:          plainTerms,
			ExcludedTerms:  excludedPlainTerms,
			TermsType:      TERMS_TYPE_PLAIN,
			PostType:       postType,
			User:           user,
			Parent:         parent,
			Ids:            ids,
			MinVotes:       minVotes,
			MaxVotes:       maxVotes,
			MinAnswers:     minAnswers,
			MaxAnswers:     maxAnswers,
			FromDate:       fromDate,
			ToDate:         toDate,
			TimeZoneOffset: timeZoneOffset,
		})
	}

	if len(tagTerms) > 0 || len(excludedTagTerms) > 0 {
		paramsList = append(paramsList, &SearchParams{
			Terms:          tagTerms,
			ExcludedTerms:  excludedTagTerms,
			TermsType:      TERMS_TYPE_TAG,
			PostType:       postType,
			User:           user,
			Parent:         parent,
			Ids:            ids,
			MinVotes:       minVotes,
			MaxVotes:       maxVotes,
			MinAnswers:     minAnswers,
			MaxAnswers:     maxAnswers,
			FromDate:       fromDate,
			ToDate:         toDate,
			TimeZoneOffset: timeZoneOffset,
		})
	}

	if len(bodyTerms) > 0 || len(excludedBodyTerms) > 0 {
		paramsList = append(paramsList, &SearchParams{
			Terms:          bodyTerms,
			ExcludedTerms:  excludedBodyTerms,
			TermsType:      TERMS_TYPE_BODY,
			PostType:       postType,
			User:           user,
			Parent:         parent,
			Ids:            ids,
			MinVotes:       minVotes,
			MaxVotes:       maxVotes,
			MinAnswers:     minAnswers,
			MaxAnswers:     maxAnswers,
			FromDate:       fromDate,
			ToDate:         toDate,
			TimeZoneOffset: timeZoneOffset,
		})
	}

	if len(titleTerms) > 0 || len(excludedTitleTerms) > 0 {
		paramsList = append(paramsList, &SearchParams{
			Terms:          titleTerms,
			ExcludedTerms:  excludedTitleTerms,
			TermsType:      TERMS_TYPE_TITLE,
			PostType:       postType,
			User:           user,
			Parent:         parent,
			Ids:            ids,
			MinVotes:       minVotes,
			MaxVotes:       maxVotes,
			MinAnswers:     minAnswers,
			MaxAnswers:     maxAnswers,
			FromDate:       fromDate,
			ToDate:         toDate,
			TimeZoneOffset: timeZoneOffset,
		})
	}

	if len(paramsList) == 0 &&
		(len(postType) != 0 || len(user) != 0 ||
			len(parent) != 0 || len(ids) != 0 ||
			minVotes != nil || maxVotes != nil ||
			len(fromDate) != 0 || len(toDate) != 0 ||
			minAnswers != nil || maxAnswers != nil) {
		paramsList = append(paramsList, &SearchParams{
			Terms:          "",
			ExcludedTerms:  "",
			TermsType:      "",
			PostType:       postType,
			User:           user,
			Parent:         parent,
			Ids:            ids,
			MinVotes:       minVotes,
			MaxVotes:       maxVotes,
			MinAnswers:     minAnswers,
			MaxAnswers:     maxAnswers,
			FromDate:       fromDate,
			ToDate:         toDate,
			TimeZoneOffset: timeZoneOffset,
		})
	}

	return paramsList
}
