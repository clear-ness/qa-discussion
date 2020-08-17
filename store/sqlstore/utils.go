package sqlstore

import (
	"bytes"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/clear-ness/qa-discussion/mlog"
	"github.com/go-gorp/gorp"
)

func finalizeTransaction(transaction *gorp.Transaction) {
	if err := transaction.Rollback(); err != nil && err != sql.ErrTxDone {
		mlog.Error("Failed to rollback transaction", mlog.Err(err))
	}
}

func MapStringsToQueryParams(list []string, paramPrefix string) (string, map[string]interface{}) {
	keys := bytes.Buffer{}
	params := make(map[string]interface{}, len(list))
	if len(list) == 0 {
		return "('')", params
	}

	for i, entry := range list {
		if keys.Len() > 0 {
			keys.WriteString(",")
		}

		key := paramPrefix + strconv.Itoa(i)
		keys.WriteString(":" + key)
		params[key] = entry
	}

	return fmt.Sprintf("(%v)", keys.String()), params
}

var escapeLikeSearchChar = []string{
	"%",
	"_",
}

func sanitizeSearchTerm(term string, escapeChar string) string {
	term = strings.Replace(term, escapeChar, "", -1)

	for _, c := range escapeLikeSearchChar {
		term = strings.Replace(term, c, escapeChar+c, -1)
	}

	return term
}
