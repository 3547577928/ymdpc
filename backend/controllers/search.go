package controllers

import (
	"log/slog"
	"strings"
	"sync/atomic"
	"unicode/utf8"

	"gorm.io/gorm"
)

// postSearchFTS 标识全文索引是否可用：驱动不支持或建索引失败时静默退化为 LIKE
var postSearchFTS atomic.Bool

// SetupPostSearch 建立文章全文索引。使用 trigram tokenizer：unicode61 对中文
// 不做分词（整段当一个词，无法子串匹配），trigram 以三字滑窗索引，中英文都能
// 子串命中，代价是匹配至少 3 个字符（更短的词由调用方回退 LIKE）。
// 启动时 rebuild 一次保证索引与 posts 表一致。
func SetupPostSearch(db *gorm.DB) {
	statements := []string{
		`CREATE VIRTUAL TABLE IF NOT EXISTS posts_fts USING fts5(title, summary, content, content='posts', content_rowid='id', tokenize='trigram')`,
		`CREATE TRIGGER IF NOT EXISTS posts_fts_ai AFTER INSERT ON posts BEGIN
			INSERT INTO posts_fts(rowid, title, summary, content) VALUES (new.id, new.title, new.summary, new.content);
		END`,
		`CREATE TRIGGER IF NOT EXISTS posts_fts_ad AFTER DELETE ON posts BEGIN
			INSERT INTO posts_fts(posts_fts, rowid, title, summary, content) VALUES ('delete', old.id, old.title, old.summary, old.content);
		END`,
		`CREATE TRIGGER IF NOT EXISTS posts_fts_au AFTER UPDATE ON posts BEGIN
			INSERT INTO posts_fts(posts_fts, rowid, title, summary, content) VALUES ('delete', old.id, old.title, old.summary, old.content);
			INSERT INTO posts_fts(rowid, title, summary, content) VALUES (new.id, new.title, new.summary, new.content);
		END`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			slog.Warn("full-text search unavailable, falling back to LIKE", "err", err)
			return
		}
	}
	if err := db.Exec("INSERT INTO posts_fts(posts_fts) VALUES ('rebuild')").Error; err != nil {
		slog.Warn("rebuild full-text index failed, falling back to LIKE", "err", err)
		return
	}
	postSearchFTS.Store(true)
}

// ftsMatchExpression 把用户输入转成 FTS5 MATCH 表达式：按空白分词后逐词加双引号，
// 防止用户输入的引号/操作符注入 MATCH 语法；词间为 AND 语义
func ftsMatchExpression(keyword string) string {
	quoted := make([]string, 0, 4)
	for _, term := range strings.Fields(keyword) {
		// trigram 索引最少 3 个字符才能匹配，过短的词交给 LIKE 兜底
		if utf8.RuneCountInString(term) < 3 {
			continue
		}
		quoted = append(quoted, `"`+strings.ReplaceAll(term, `"`, `""`)+`"`)
	}
	return strings.Join(quoted, " ")
}

// postSearchCondition 为文章关键字搜索添加 WHERE 条件：优先全文索引，
// 索引不可用或关键字全部过短时退化为 LIKE（行为与旧版一致）
func postSearchCondition(query *gorm.DB, keyword string) *gorm.DB {
	if postSearchFTS.Load() {
		if match := ftsMatchExpression(keyword); match != "" {
			return query.Where("posts.id IN (SELECT rowid FROM posts_fts WHERE posts_fts MATCH ?)", match)
		}
	}
	like := likePattern(keyword)
	return query.Where("posts.title LIKE ? ESCAPE '\\' OR posts.summary LIKE ? ESCAPE '\\' OR posts.content LIKE ? ESCAPE '\\'", like, like, like)
}
