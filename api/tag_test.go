package api

import (
	"strconv"
	"strings"
	"testing"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutocompleteTags(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	question := &model.Post{}
	question.Title = "title1"
	question.Content = "question content"

	tags := ""
	for i := 0; i < model.MAX_PARSE_TAG_COUNT; i++ {
		tags = tags + "tag" + strconv.Itoa(i)
		if i < (model.MAX_PARSE_TAG_COUNT - 1) {
			tags = tags + " "
		}
	}

	question.Tags = tags
	rpost, resp := Client.CreateQuestion(question)
	CheckNoError(t, resp)
	assert.Equal(t, tags, rpost.Tags, "failed to create tags")

	data, resp := Client.AutocompleteTags("")
	CheckNoError(t, resp)
	require.Len(t, data, 0, "invalid tags")

	data, resp = Client.AutocompleteTags("t")
	CheckNoError(t, resp)
	require.Len(t, data, model.MAX_PARSE_TAG_COUNT, "invalid tags")
	for i := 0; i < model.MAX_PARSE_TAG_COUNT; i++ {
		require.Equal(t, data[i].Content, strings.Fields(tags)[i], "autocomplete tags didn't match")
	}

	data, resp = Client.AutocompleteTags(strings.Fields(tags)[0])
	CheckNoError(t, resp)
	require.Len(t, data, 1, "invalid tags")
}
