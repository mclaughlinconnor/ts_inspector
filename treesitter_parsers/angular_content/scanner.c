#include "parser.h"
#include <assert.h>
#include <ctype.h>
#include <stdio.h>
#include <string.h>

enum TokenType {
  CONTENT,
  TEXT,
};

typedef struct {
  bool parens;
} Scanner;

static inline void advance(TSLexer *lexer) { lexer->advance(lexer, false); }

static inline void skip(TSLexer *lexer) { lexer->advance(lexer, true); }

unsigned tree_sitter_angular_content_external_scanner_serialize(void *payload,
                                                                char *buffer) {
  Scanner *scanner = (Scanner *)payload;
  size_t size = 0;

  buffer[size++] = (char)scanner->parens;

  return size;
}

void tree_sitter_angular_content_external_scanner_deserialize(
    void *payload, const char *buffer, unsigned length) {
  Scanner *scanner = (Scanner *)payload;
  scanner->parens = false;

  if (length > 0) {
    (scanner->parens = (unsigned char)buffer[0]);
  }
}

void *tree_sitter_angular_content_external_scanner_create() {
  Scanner *scanner = calloc(1, sizeof(Scanner));

  scanner->parens = false;

  tree_sitter_angular_content_external_scanner_deserialize(scanner, NULL, 0);

  return scanner;
}

bool tree_sitter_angular_content_external_scanner_scan(
    void *payload, TSLexer *lexer, const bool *valid_symbols) {
  Scanner *scanner = (Scanner *)payload;

  if (lexer->eof(lexer)) {
    return false;
  }

  // An empty {{}}
  if (!scanner->parens && lexer->lookahead == '}') {
    lexer->result_symbol = CONTENT;
    return true;
  }

  // Returned true from CONTENT, then the parser retries CONTENT
  if (scanner->parens && lexer->lookahead == '}') {
    scanner->parens = false;

    return false;
  }

  if (valid_symbols[CONTENT]) {
    lexer->result_symbol = CONTENT;
    scanner->parens = false;

    for (;;) {
      advance(lexer);
      while (lexer->lookahead != '}' && !lexer->eof(lexer)) {
        advance(lexer);
      }

      lexer->mark_end(lexer);

      if (lexer->eof(lexer)) {
        lexer->mark_end(lexer);
        lexer->result_symbol = CONTENT;
        return true;
      }

      skip(lexer);
      if (lexer->lookahead != '}') {
        continue;
      }

      lexer->result_symbol = CONTENT;
      return true;
    }

    return true;
  }

  if (valid_symbols[TEXT] && lexer->lookahead != '\0') {

    while (lexer->lookahead != '{' && !lexer->eof(lexer)) {
      lexer->result_symbol = TEXT;
      advance(lexer);
      lexer->mark_end(lexer);
    }

    return lexer->result_symbol == TEXT;
  }

  return false;
}

void tree_sitter_angular_content_external_scanner_destroy(void *payload) {
  Scanner *scanner = (Scanner *)payload;
  free(scanner);
}
