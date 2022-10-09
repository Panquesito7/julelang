package ast

import (
	"os"
	"strings"
	"sync"

	"github.com/jule-lang/jule/ast/models"
	"github.com/jule-lang/jule/lex"
	"github.com/jule-lang/jule/pkg/jule"
	"github.com/jule-lang/jule/pkg/juleapi"
	"github.com/jule-lang/jule/pkg/julelog"
	"github.com/jule-lang/jule/pkg/juletype"
)

// Builder is builds AST tree.
type Builder struct {
	wg  sync.WaitGroup
	pub bool

	Tree   []models.Object
	Errors []julelog.CompilerLog
	Tokens []lex.Token
	Pos    int
}

// NewBuilder instance.
func NewBuilder(t []lex.Token) *Builder {
	b := new(Builder)
	b.Tokens = t
	b.Pos = 0
	return b
}

func compilerErr(t lex.Token, key string, args ...any) julelog.CompilerLog {
	return julelog.CompilerLog{
		Type:    julelog.ERR,
		Row:     t.Row,
		Column:  t.Column,
		Path:    t.File.Path(),
		Message: jule.GetError(key, args...),
	}
}

// pusherr appends error by specified token.
func (b *Builder) pusherr(t lex.Token, key string, args ...any) {
	b.Errors = append(b.Errors, compilerErr(t, key, args...))
}

// Ended reports position is at end of tokens or not.
func (b *Builder) Ended() bool {
	return b.Pos >= len(b.Tokens)
}

func (b *Builder) buildNode(toks []lex.Token) {
	t := toks[0]
	switch t.Id {
	case lex.ID_USE:
		b.Use(toks)
	case lex.ID_FN, lex.ID_UNSAFE:
		s := models.Statement{Token: t}
		s.Data = b.Func(toks, false, false, false)
		b.Tree = append(b.Tree, models.Object{Token: s.Token, Data: s})
	case lex.ID_CONST, lex.ID_LET, lex.ID_MUT:
		b.GlobalVar(toks)
	case lex.ID_TYPE:
		b.Tree = append(b.Tree, b.TypeOrGenerics(toks))
	case lex.ID_ENUM:
		b.Enum(toks)
	case lex.ID_STRUCT:
		b.Struct(toks)
	case lex.ID_TRAIT:
		b.Trait(toks)
	case lex.ID_IMPL:
		b.Impl(toks)
	case lex.ID_CPP:
		b.CppLink(toks)
	case lex.ID_COMMENT:
		b.Tree = append(b.Tree, b.Comment(toks[0]))
	default:
		b.pusherr(t, "invalid_syntax")
		return
	}
	if b.pub {
		b.pusherr(t, "def_not_support_pub")
	}
}

// Build builds AST tree.
func (b *Builder) Build() {
	for b.Pos != -1 && !b.Ended() {
		toks := b.nextBuilderStatement()
		b.pub = toks[0].Id == lex.ID_PUB
		if b.pub {
			if len(toks) == 1 {
				if b.Ended() {
					b.pusherr(toks[0], "invalid_syntax")
					continue
				}
				toks = b.nextBuilderStatement()
			} else {
				toks = toks[1:]
			}
		}
		b.buildNode(toks)
	}
	b.Wait()
}

// Wait waits for concurrency.
func (b *Builder) Wait() { b.wg.Wait() }

// TypeAlias builds AST model of type definition statement.
func (b *Builder) TypeAlias(toks []lex.Token) (t models.TypeAlias) {
	i := 1 // Initialize value is 1 for skip keyword.
	if i >= len(toks) {
		b.pusherr(toks[i-1], "invalid_syntax")
		return
	}
	t.Token = toks[1]
	t.Id = t.Token.Kind
	token := toks[i]
	if token.Id != lex.ID_IDENT {
		b.pusherr(token, "invalid_syntax")
	}
	i++
	if i >= len(toks) {
		b.pusherr(toks[i-1], "invalid_syntax")
		return
	}
	token = toks[i]
	if token.Id != lex.ID_COLON {
		b.pusherr(toks[i-1], "invalid_syntax")
		return
	}
	i++
	if i >= len(toks) {
		b.pusherr(toks[i-1], "missing_type")
		return
	}
	destType, ok := b.DataType(toks, &i, true, true)
	t.Type = destType
	if ok && i+1 < len(toks) {
		b.pusherr(toks[i+1], "invalid_syntax")
	}
	return
}

func (b *Builder) buildEnumItemExpr(i *int, toks []lex.Token) models.Expr {
	brace_n := 0
	exprStart := *i
	for ; *i < len(toks); *i++ {
		t := toks[*i]
		if t.Id == lex.ID_BRACE {
			switch t.Kind {
			case lex.KND_LBRACE, lex.KND_LBRACKET, lex.KND_LPAREN:
				brace_n++
				continue
			default:
				brace_n--
			}
		}
		if brace_n > 0 {
			continue
		}
		if t.Id == lex.ID_COMMA || *i+1 >= len(toks) {
			var exprToks []lex.Token
			if t.Id == lex.ID_COMMA {
				exprToks = toks[exprStart:*i]
			} else {
				exprToks = toks[exprStart:]
			}
			return b.Expr(exprToks)
		}
	}
	return models.Expr{}
}

func (b *Builder) buildEnumItems(toks []lex.Token) []*models.EnumItem {
	items := make([]*models.EnumItem, 0)
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		if t.Id == lex.ID_COMMENT {
			continue
		}
		item := new(models.EnumItem)
		item.Token = t
		if item.Token.Id != lex.ID_IDENT {
			b.pusherr(item.Token, "invalid_syntax")
		}
		item.Id = item.Token.Kind
		if i+1 >= len(toks) || toks[i+1].Id == lex.ID_COMMA {
			if i+1 < len(toks) {
				i++
			}
			items = append(items, item)
			continue
		}
		i++
		t = toks[i]
		if t.Id != lex.ID_OP && t.Kind != lex.KND_EQ {
			b.pusherr(toks[0], "invalid_syntax")
		}
		i++
		if i >= len(toks) || toks[i].Id == lex.ID_COMMA {
			b.pusherr(toks[0], "missing_expr")
			continue
		}
		item.Expr = b.buildEnumItemExpr(&i, toks)
		items = append(items, item)
	}
	return items
}

// Enum builds AST model of enumerator statement.
func (b *Builder) Enum(toks []lex.Token) {
	var e models.Enum
	if len(toks) < 2 || len(toks) < 3 {
		b.pusherr(toks[0], "invalid_syntax")
		return
	}
	e.Token = toks[1]
	if e.Token.Id != lex.ID_IDENT {
		b.pusherr(e.Token, "invalid_syntax")
	}
	e.Id = e.Token.Kind
	i := 2
	if toks[i].Id == lex.ID_COLON {
		i++
		if i >= len(toks) {
			b.pusherr(toks[i-1], "invalid_syntax")
			return
		}
		e.Type, _ = b.DataType(toks, &i, false, true)
		i++
		if i >= len(toks) {
			b.pusherr(e.Token, "body_not_exist")
			return
		}
	} else {
		e.Type = models.Type{Id: juletype.U32, Kind: juletype.TYPE_MAP[juletype.U32]}
	}
	itemToks := b.getrange(&i, lex.KND_LBRACE, lex.KND_RBRACE, &toks)
	if itemToks == nil {
		b.pusherr(e.Token, "body_not_exist")
		return
	} else if i < len(toks) {
		b.pusherr(toks[i], "invalid_syntax")
	}
	e.Pub = b.pub
	b.pub = false
	e.Items = b.buildEnumItems(itemToks)
	b.Tree = append(b.Tree, models.Object{Token: e.Token, Data: e})
}

// Comment builds AST model of comment.
func (b *Builder) Comment(t lex.Token) models.Object {
	t.Kind = strings.TrimSpace(t.Kind[2:])
	return models.Object{
		Token: t,
		Data: models.Comment{
			Token:   t,
			Content: t.Kind,
		},
	}
}

func (b *Builder) structFields(toks []lex.Token, cpp_linked bool) []*models.Var {
	var fields []*models.Var
	i := 0
	for i < len(toks) {
		var_tokens := b.skipStatement(&i, &toks)
		if var_tokens[0].Id == lex.ID_COMMENT {
			continue
		}
		is_pub := var_tokens[0].Id == lex.ID_PUB
		if is_pub {
			if len(var_tokens) == 1 {
				b.pusherr(var_tokens[0], "invalid_syntax")
				continue
			}
			var_tokens = var_tokens[1:]
		}
		is_mut := var_tokens[0].Id == lex.ID_MUT
		if is_mut {
			if len(var_tokens) == 1 {
				b.pusherr(var_tokens[0], "invalid_syntax")
				continue
			}
			var_tokens = var_tokens[1:]
		}
		v := b.Var(var_tokens, false, false)
		v.Pub = is_pub
		v.Mutable = is_mut
		v.IsField = true
		v.CppLinked = cpp_linked
		fields = append(fields, &v)
	}
	return fields
}

func (b *Builder) parse_struct(toks []lex.Token, cpp_linked bool) models.Struct {
	var s models.Struct
	s.Pub = b.pub
	b.pub = false
	if len(toks) < 3 {
		b.pusherr(toks[0], "invalid_syntax")
		return s
	}
	s.Token = toks[1]
	if s.Token.Id != lex.ID_IDENT {
		b.pusherr(s.Token, "invalid_syntax")
	}
	s.Id = s.Token.Kind
	i := 2
	bodyToks := b.getrange(&i, lex.KND_LBRACE, lex.KND_RBRACE, &toks)
	if bodyToks == nil {
		b.pusherr(s.Token, "body_not_exist")
		return s
	}
	if i < len(toks) {
		b.pusherr(toks[i], "invalid_syntax")
	}
	s.Fields = b.structFields(bodyToks, cpp_linked)
	return s
}

// Struct builds AST model of structure.
func (b *Builder) Struct(toks []lex.Token) {
	s := b.parse_struct(toks, false)
	b.Tree = append(b.Tree, models.Object{Token: s.Token, Data: s})
}

func (b *Builder) traitFuncs(toks []lex.Token, trait_id string) []*models.Fn {
	var funcs []*models.Fn
	i := 0
	for i < len(toks) {
		fnToks := b.skipStatement(&i, &toks)
		f := b.Func(fnToks, true, false, true)
		b.setup_receiver(&f, trait_id)
		f.Pub = true
		funcs = append(funcs, &f)
	}
	return funcs
}

// Trait builds AST model of trait.
func (b *Builder) Trait(toks []lex.Token) {
	var t models.Trait
	t.Pub = b.pub
	b.pub = false
	if len(toks) < 3 {
		b.pusherr(toks[0], "invalid_syntax")
		return
	}
	t.Token = toks[1]
	if t.Token.Id != lex.ID_IDENT {
		b.pusherr(t.Token, "invalid_syntax")
	}
	t.Id = t.Token.Kind
	i := 2
	bodyToks := b.getrange(&i, lex.KND_LBRACE, lex.KND_RBRACE, &toks)
	if bodyToks == nil {
		b.pusherr(t.Token, "body_not_exist")
		return
	}
	if i < len(toks) {
		b.pusherr(toks[i], "invalid_syntax")
	}
	t.Funcs = b.traitFuncs(bodyToks, t.Id)
	b.Tree = append(b.Tree, models.Object{Token: t.Token, Data: t})
}

func (b *Builder) implTraitFuncs(impl *models.Impl, toks []lex.Token) {
	pos, btoks := b.Pos, make([]lex.Token, len(b.Tokens))
	copy(btoks, b.Tokens)
	defer func() { b.Pos, b.Tokens = pos, btoks }()
	b.Pos = 0
	b.Tokens = toks
	for b.Pos != -1 && !b.Ended() {
		fnToks := b.nextBuilderStatement()
		tok := fnToks[0]
		switch tok.Id {
		case lex.ID_COMMENT:
			impl.Tree = append(impl.Tree, b.Comment(tok))
			continue
		case lex.ID_FN, lex.ID_UNSAFE:
			f := b.get_method(fnToks)
			f.Pub = true
			b.setup_receiver(f, impl.Target.Kind)
			impl.Tree = append(impl.Tree, models.Object{Token: f.Token, Data: f})
		default:
			b.pusherr(tok, "invalid_syntax")
			continue
		}
	}
}

func (b *Builder) implStruct(impl *models.Impl, toks []lex.Token) {
	pos, btoks := b.Pos, make([]lex.Token, len(b.Tokens))
	copy(btoks, b.Tokens)
	defer func() { b.Pos, b.Tokens = pos, btoks }()
	b.Pos = 0
	b.Tokens = toks
	for b.Pos != -1 && !b.Ended() {
		fnToks := b.nextBuilderStatement()
		tok := fnToks[0]
		pub := false
		switch tok.Id {
		case lex.ID_COMMENT:
			impl.Tree = append(impl.Tree, b.Comment(tok))
			continue
		case lex.ID_TYPE:
			impl.Tree = append(impl.Tree, models.Object{
				Token: tok,
				Data:  b.Generics(fnToks),
			})
			continue
		}
		if tok.Id == lex.ID_PUB {
			pub = true
			if len(fnToks) == 1 {
				b.pusherr(fnToks[0], "invalid_syntax")
				continue
			}
			fnToks = fnToks[1:]
			if len(fnToks) > 0 {
				tok = fnToks[0]
			}
		}
		switch tok.Id {
		case lex.ID_FN, lex.ID_UNSAFE:
			f := b.get_method(fnToks)
			f.Pub = pub
			b.setup_receiver(f, impl.Base.Kind)
			impl.Tree = append(impl.Tree, models.Object{Token: f.Token, Data: f})
		default:
			b.pusherr(tok, "invalid_syntax")
			continue
		}
	}
}

func (b *Builder) get_method(toks []lex.Token) *models.Fn {
	tok := toks[0]
	if tok.Id == lex.ID_UNSAFE {
		toks = toks[1:]
		if len(toks) == 0 || toks[0].Id != lex.ID_FN {
			b.pusherr(tok, "invalid_syntax")
			return nil
		}
	} else if toks[0].Id != lex.ID_FN {
		b.pusherr(tok, "invalid_syntax")
		return nil
	}
	f := new(models.Fn)
	*f = b.Func(toks, true, false, false)
	f.IsUnsafe = tok.Id == lex.ID_UNSAFE
	if f.Block != nil {
		f.Block.IsUnsafe = f.IsUnsafe
	}
	return f
}

func (b *Builder) implFuncs(impl *models.Impl, toks []lex.Token) {
	if impl.Target.Id != juletype.VOID {
		b.implTraitFuncs(impl, toks)
		return
	}
	b.implStruct(impl, toks)
}

// Impl builds AST model of impl statement.
func (b *Builder) Impl(toks []lex.Token) {
	tok := toks[0]
	if len(toks) < 2 {
		b.pusherr(tok, "invalid_syntax")
		return
	}
	tok = toks[1]
	if tok.Id != lex.ID_IDENT {
		b.pusherr(tok, "invalid_syntax")
		return
	}
	var impl models.Impl
	if len(toks) < 3 {
		b.pusherr(tok, "invalid_syntax")
		return
	}
	impl.Base = tok
	tok = toks[2]
	if tok.Id != lex.ID_ITER {
		if tok.Id == lex.ID_BRACE && tok.Kind == lex.KND_LBRACE {
			toks = toks[2:]
			goto body
		}
		b.pusherr(tok, "invalid_syntax")
		return
	}
	if len(toks) < 4 {
		b.pusherr(tok, "invalid_syntax")
		return
	}
	tok = toks[3]
	if tok.Id != lex.ID_IDENT {
		b.pusherr(tok, "invalid_syntax")
		return
	}
	{
		i := 0
		impl.Target, _ = b.DataType(toks[3:4], &i, false, true)
		toks = toks[4:]
	}
body:
	i := 0
	bodyToks := b.getrange(&i, lex.KND_LBRACE, lex.KND_RBRACE, &toks)
	if bodyToks == nil {
		b.pusherr(impl.Base, "body_not_exist")
		return
	}
	if i < len(toks) {
		b.pusherr(toks[i], "invalid_syntax")
	}
	b.implFuncs(&impl, bodyToks)
	b.Tree = append(b.Tree, models.Object{Token: impl.Base, Data: impl})
}

// link_fn builds AST model of cpp function link.
func (b *Builder) link_fn(toks []lex.Token) {
	tok := toks[0]

	// Catch pub not supported
	bpub := b.pub
	b.pub = false

	var link models.CppLinkFn
	link.Token = tok
	link.Link = new(models.Fn)
	*link.Link = b.Func(toks[1:], false, false, true)
	b.Tree = append(b.Tree, models.Object{Token: tok, Data: link})

	b.pub = bpub
}

// link_var builds AST model of cpp variable link.
func (b *Builder) link_var(toks []lex.Token) {
	tok := toks[0]

	// Catch pub not supported
	bpub := b.pub
	b.pub = false

	var link models.CppLinkVar
	link.Token = tok
	link.Link = new(models.Var)
	*link.Link = b.Var(toks[1:], true, false)
	b.Tree = append(b.Tree, models.Object{Token: tok, Data: link})

	b.pub = bpub
}

// link_struct builds AST model of cpp structure link.
func (b *Builder) link_struct(toks []lex.Token) {
	tok := toks[0]

	// Catch pub not supported
	bpub := b.pub
	b.pub = false

	var link models.CppLinkStruct
	link.Token = tok
	link.Link = b.parse_struct(toks[1:], true)
	b.Tree = append(b.Tree, models.Object{Token: tok, Data: link})

	b.pub = bpub
}

// link_type_alias builds AST model of cpp type alias link.
func (b *Builder) link_type_alias(toks []lex.Token) {
	tok := toks[0]

	// Catch pub not supported
	bpub := b.pub
	b.pub = false

	var link models.CppLinkAlias
	link.Token = tok
	link.Link = b.TypeAlias(toks[1:])
	b.Tree = append(b.Tree, models.Object{Token: tok, Data: link})

	b.pub = bpub
}

// CppLinks builds AST model of cpp link statement.
func (b *Builder) CppLink(toks []lex.Token) {
	tok := toks[0]
	if len(toks) == 1 {
		b.pusherr(tok, "invalid_syntax")
		return
	}
	tok = toks[1]
	switch tok.Id {
	case lex.ID_FN, lex.ID_UNSAFE:
		b.link_fn(toks)
	case lex.ID_LET:
		b.link_var(toks)
	case lex.ID_STRUCT:
		b.link_struct(toks)
	case lex.ID_TYPE:
		b.link_type_alias(toks)
	default:
		b.pusherr(tok, "invalid_syntax")
	}
}

func tokstoa(toks []lex.Token) string {
	var str strings.Builder
	for _, tok := range toks {
		str.WriteString(tok.Kind)
	}
	return str.String()
}

// Use builds AST model of use declaration.
func (b *Builder) Use(toks []lex.Token) {
	var use models.UseDecl
	use.Token = toks[0]
	if len(toks) < 2 {
		b.pusherr(use.Token, "missing_use_path")
		return
	}
	toks = toks[1:]
	b.buildUseDecl(&use, toks)
	b.Tree = append(b.Tree, models.Object{Token: use.Token, Data: use})
}

func (b *Builder) getSelectors(toks []lex.Token) []lex.Token {
	i := 0
	toks = b.getrange(&i, lex.KND_LBRACE, lex.KND_RBRACE, &toks)
	parts, errs := Parts(toks, lex.ID_COMMA, true)
	if len(errs) > 0 {
		b.Errors = append(b.Errors, errs...)
		return nil
	}
	selectors := make([]lex.Token, len(parts))
	for i, part := range parts {
		if len(part) > 1 {
			b.pusherr(part[1], "invalid_syntax")
		}
		tok := part[0]
		if tok.Id != lex.ID_IDENT && tok.Id != lex.ID_SELF {
			b.pusherr(tok, "invalid_syntax")
			continue
		}
		selectors[i] = tok
	}
	return selectors
}

func (b *Builder) buildUseCppDecl(use *models.UseDecl, toks []lex.Token) {
	if len(toks) > 2 {
		b.pusherr(toks[2], "invalid_syntax")
	}
	tok := toks[1]
	if tok.Id != lex.ID_LITERAL || (tok.Kind[0] != '`' && tok.Kind[0] != '"') {
		b.pusherr(tok, "invalid_expr")
		return
	}
	use.Cpp = true
	use.Path = tok.Kind[1 : len(tok.Kind)-1]
}

func (b *Builder) buildUseDecl(use *models.UseDecl, toks []lex.Token) {
	var path strings.Builder
	path.WriteString(jule.STDLIB_PATH)
	path.WriteRune(os.PathSeparator)
	tok := toks[0]
	isStd := false
	if tok.Id == lex.ID_CPP {
		b.buildUseCppDecl(use, toks)
		return
	}
	if tok.Id != lex.ID_IDENT || tok.Kind != "std" {
		b.pusherr(toks[0], "invalid_syntax")
	}
	isStd = true
	if len(toks) < 3 {
		b.pusherr(tok, "invalid_syntax")
		return
	}
	toks = toks[2:]
	tok = toks[len(toks)-1]
	switch tok.Id {
	case lex.ID_DBLCOLON:
		b.pusherr(tok, "invalid_syntax")
		return
	case lex.ID_BRACE:
		if tok.Kind != lex.KND_RBRACE {
			b.pusherr(tok, "invalid_syntax")
			return
		}
		var selectors []lex.Token
		toks, selectors = RangeLast(toks)
		use.Selectors = b.getSelectors(selectors)
		if len(toks) == 0 {
			b.pusherr(tok, "invalid_syntax")
			return
		}
		tok = toks[len(toks)-1]
		if tok.Id != lex.ID_DBLCOLON {
			b.pusherr(tok, "invalid_syntax")
			return
		}
		toks = toks[:len(toks)-1]
		if len(toks) == 0 {
			b.pusherr(tok, "invalid_syntax")
			return
		}
	case lex.ID_OP:
		if tok.Kind != lex.KND_STAR {
			b.pusherr(tok, "invalid_syntax")
			return
		}
		toks = toks[:len(toks)-1]
		if len(toks) == 0 {
			b.pusherr(tok, "invalid_syntax")
			return
		}
		tok = toks[len(toks)-1]
		if tok.Id != lex.ID_DBLCOLON {
			b.pusherr(tok, "invalid_syntax")
			return
		}
		toks = toks[:len(toks)-1]
		if len(toks) == 0 {
			b.pusherr(tok, "invalid_syntax")
			return
		}
		use.FullUse = true
	}
	for i, tok := range toks {
		if i%2 != 0 {
			if tok.Id != lex.ID_DBLCOLON {
				b.pusherr(tok, "invalid_syntax")
			}
			path.WriteRune(os.PathSeparator)
			continue
		}
		if tok.Id != lex.ID_IDENT {
			b.pusherr(tok, "invalid_syntax")
		}
		path.WriteString(tok.Kind)
	}
	use.LinkString = tokstoa(toks)
	if isStd {
		use.LinkString = "std::" + use.LinkString
	}
	use.Path = path.String()
}

// Attribute builds AST model of attribute.
func (b *Builder) Attribute(toks []lex.Token) (a models.Attribute) {
	i := 0
	a.Token = toks[i]
	i++
	tag := toks[i]
	if tag.Id != lex.ID_IDENT || a.Token.Column+1 != tag.Column {
		b.pusherr(tag, "invalid_syntax")
		return
	}
	a.Tag = tag.Kind
	toks = toks[i+1:]
	if len(toks) > 0 {
		tok := toks[0]
		if a.Token.Column+len(a.Tag)+1 == tok.Column {
			b.pusherr(tok, "invalid_syntax")
		}
		b.Tokens = append(toks, b.Tokens...)
	}
	return
}

func (b *Builder) setup_receiver(f *models.Fn, owner_id string) {
	if len(f.Params) == 0 {
		b.pusherr(f.Token, "missing_receiver")
		return
	}
	param := f.Params[0]
	if param.Id != lex.KND_SELF {
		b.pusherr(f.Token, "missing_receiver")
		return
	}
	f.Receiver = new(models.Var)
	f.Receiver.Type = models.Type{
		Id:   juletype.STRUCT,
		Kind: owner_id,
	}
	f.Receiver.Mutable = param.Mutable
	if param.Type.Kind != "" && param.Type.Kind[0] == '&' {
		f.Receiver.Type.Kind = lex.KND_AMPER + f.Receiver.Type.Kind
	}
	f.Params = f.Params[1:]
}

func (b *Builder) funcPrototype(toks []lex.Token, i *int, method, anon bool) (f models.Fn, ok bool) {
	ok = true
	f.Token = toks[*i]
	if f.Token.Id == lex.ID_UNSAFE {
		f.IsUnsafe = true
		*i++
		if *i >= len(toks) {
			b.pusherr(f.Token, "invalid_syntax")
			ok = false
			return
		}
		f.Token = toks[*i]
	}
	// Skips fn tok
	*i++
	if *i >= len(toks) {
		b.pusherr(f.Token, "invalid_syntax")
		ok = false
		return
	}
	f.Pub = b.pub
	b.pub = false
	if anon {
		f.Id = jule.ANONYMOUS
	} else {
		tok := toks[*i]
		if tok.Id != lex.ID_IDENT {
			b.pusherr(tok, "invalid_syntax")
			ok = false
		}
		f.Id = tok.Kind
		*i++
	}
	f.RetType.Type.Id = juletype.VOID
	f.RetType.Type.Kind = juletype.TYPE_MAP[f.RetType.Type.Id]
	if *i >= len(toks) {
		b.pusherr(f.Token, "invalid_syntax")
		return
	} else if toks[*i].Kind != lex.KND_LPAREN {
		b.pusherr(toks[*i], "missing_function_parentheses")
		return
	}
	paramToks := b.getrange(i, lex.KND_LPAREN, lex.KND_RPARENT, &toks)
	if len(paramToks) > 0 {
		f.Params = b.Params(paramToks, method, false)
	}
	t, retok := b.FuncRetDataType(toks, i)
	if retok {
		f.RetType = t
		*i++
	}
	return
}

// Func builds AST model of function.
func (b *Builder) Func(toks []lex.Token, method, anon, prototype bool) (f models.Fn) {
	var ok bool
	i := 0
	f, ok = b.funcPrototype(toks, &i, method, anon)
	if prototype {
		if i+1 < len(toks) {
			b.pusherr(toks[i+1], "invalid_syntax")
		}
		return
	} else if !ok {
		return
	}
	if i >= len(toks) {
		if b.Ended() {
			b.pusherr(f.Token, "body_not_exist")
			return
		}
		toks = b.nextBuilderStatement()
		i = 0
	}
	blockToks := b.getrange(&i, lex.KND_LBRACE, lex.KND_RBRACE, &toks)
	if blockToks != nil {
		f.Block = b.Block(blockToks)
		f.Block.IsUnsafe = f.IsUnsafe
		if i < len(toks) {
			b.pusherr(toks[i], "invalid_syntax")
		}
	} else {
		b.pusherr(f.Token, "body_not_exist")
		b.Tokens = append(toks, b.Tokens...)
	}
	return
}

func (b *Builder) generic(toks []lex.Token) models.GenericType {
	if len(toks) > 1 {
		b.pusherr(toks[1], "invalid_syntax")
	}
	var gt models.GenericType
	gt.Token = toks[0]
	if gt.Token.Id != lex.ID_IDENT {
		b.pusherr(gt.Token, "invalid_syntax")
	}
	gt.Id = gt.Token.Kind
	return gt
}

// Generic builds generic type.
func (b *Builder) Generics(toks []lex.Token) []models.GenericType {
	tok := toks[0]
	i := 1
	genericsToks := Range(&i, lex.KND_LBRACKET, lex.KND_RBRACKET, toks)
	if len(genericsToks) == 0 {
		b.pusherr(tok, "missing_expr")
		return make([]models.GenericType, 0)
	} else if i < len(toks) {
		b.pusherr(toks[i], "invalid_syntax")
	}
	parts, errs := Parts(genericsToks, lex.ID_COMMA, true)
	b.Errors = append(b.Errors, errs...)
	generics := make([]models.GenericType, len(parts))
	for i, part := range parts {
		if len(parts) == 0 {
			continue
		}
		generics[i] = b.generic(part)
	}
	return generics
}

// TypeOrGenerics builds type alias or generics type declaration.
func (b *Builder) TypeOrGenerics(toks []lex.Token) models.Object {
	if len(toks) > 1 {
		tok := toks[1]
		if tok.Id == lex.ID_BRACE && tok.Kind == lex.KND_LBRACKET {
			generics := b.Generics(toks)
			return models.Object{
				Token: tok,
				Data:  generics,
			}
		}
	}
	t := b.TypeAlias(toks)
	t.Pub = b.pub
	b.pub = false
	return models.Object{
		Token: t.Token,
		Data:  t,
	}
}

// GlobalVar builds AST model of global variable.
func (b *Builder) GlobalVar(toks []lex.Token) {
	if toks == nil {
		return
	}
	bs := blockStatement{toks: toks}
	s := b.VarStatement(&bs, true)
	b.Tree = append(b.Tree, models.Object{
		Token: s.Token,
		Data:  s,
	})
}

func (b *Builder) build_self(toks []lex.Token) (model models.Param) {
	if len(toks) == 0 {
		return
	}
	i := 0
	if toks[i].Id == lex.ID_MUT {
		model.Mutable = true
		i++
		if i >= len(toks) {
			b.pusherr(toks[i-1], "invalid_syntax")
			return
		}
	}
	if toks[i].Kind == lex.KND_AMPER {
		model.Type.Kind = "&"
		i++
		if i >= len(toks) {
			b.pusherr(toks[i-1], "invalid_syntax")
			return
		}
	}
	if toks[i].Id == lex.ID_SELF {
		model.Id = lex.KND_SELF
		model.Token = toks[i]
		i++
		if i < len(toks) {
			b.pusherr(toks[i+1], "invalid_syntax")
		}
	}
	return
}

// Params builds AST model of function parameters.
func (b *Builder) Params(toks []lex.Token, method, mustPure bool) []models.Param {
	parts, errs := Parts(toks, lex.ID_COMMA, true)
	b.Errors = append(b.Errors, errs...)
	if len(parts) == 0 {
		return nil
	}
	var params []models.Param
	if method && len(parts) > 0 {
		param := b.build_self(parts[0])
		if param.Id == lex.KND_SELF {
			params = append(params, param)
			parts = parts[1:]
		}
	}
	for _, part := range parts {
		b.pushParam(&params, part, mustPure)
	}
	b.checkParams(&params)
	return params
}

func (b *Builder) checkParams(params *[]models.Param) {
	for i := range *params {
		param := &(*params)[i]
		if param.Id == lex.KND_SELF || param.Type.Token.Id != lex.ID_NA {
			continue
		}
		if param.Token.Id == lex.ID_NA {
			b.pusherr(param.Token, "missing_type")
		} else {
			param.Type.Token = param.Token
			param.Type.Id = juletype.ID
			param.Type.Kind = param.Type.Token.Kind
			param.Type.Original = param.Type
			param.Id = jule.ANONYMOUS
			param.Token = lex.Token{}
		}
	}
}

func (b *Builder) paramTypeBegin(param *models.Param, i *int, toks []lex.Token) {
	for ; *i < len(toks); *i++ {
		tok := toks[*i]
		switch tok.Id {
		case lex.ID_OP:
			switch tok.Kind {
			case lex.KND_TRIPLE_DOT:
				if param.Variadic {
					b.pusherr(tok, "already_variadic")
					continue
				}
				param.Variadic = true
			default:
				return
			}
		default:
			return
		}
	}
}

func (b *Builder) paramBodyId(param *models.Param, tok lex.Token) {
	if juleapi.IsIgnoreId(tok.Kind) {
		param.Id = jule.ANONYMOUS
		return
	}
	param.Id = tok.Kind
}

func (b *Builder) paramBody(param *models.Param, i *int, toks []lex.Token, mustPure bool) {
	b.paramBodyId(param, toks[*i])
	// +1 for skip identifier token
	tok := toks[*i]
	toks = toks[*i+1:]
	if len(toks) == 0 {
		return
	} else if len(toks) < 2 {
		b.pusherr(tok, "missing_type")
		return
	}
	tok = toks[*i]
	if tok.Id != lex.ID_COLON {
		b.pusherr(tok, "invalid_syntax")
		return
	}
	toks = toks[*i+1:] // Skip colon
	b.paramType(param, toks, mustPure)
}

func (b *Builder) paramType(param *models.Param, toks []lex.Token, mustPure bool) {
	i := 0
	if !mustPure {
		b.paramTypeBegin(param, &i, toks)
		if i >= len(toks) {
			return
		}
	}
	param.Type, _ = b.DataType(toks, &i, false, true)
	i++
	if i < len(toks) {
		b.pusherr(toks[i], "invalid_syntax")
	}
}

func (b *Builder) pushParam(params *[]models.Param, toks []lex.Token, mustPure bool) {
	var param models.Param
	param.Token = toks[0]
	if param.Token.Id == lex.ID_MUT {
		param.Mutable = true
		if len(toks) == 1 {
			b.pusherr(toks[0], "invalid_syntax")
			return
		}
		toks = toks[1:]
		param.Token = toks[0]
	}
	// Just data type
	if param.Token.Id != lex.ID_IDENT {
		param.Id = jule.ANONYMOUS
		b.paramType(&param, toks, mustPure)
	} else {
		i := 0
		b.paramBody(&param, &i, toks, mustPure)
	}
	*params = append(*params, param)
}

func (b *Builder) idGenericsParts(toks []lex.Token, i *int) [][]lex.Token {
	first := *i
	brace_n := 0
	for ; *i < len(toks); *i++ {
		tok := toks[*i]
		if tok.Id == lex.ID_BRACE {
			switch tok.Kind {
			case lex.KND_LBRACKET:
				brace_n++
			case lex.KND_RBRACKET:
				brace_n--
			}
		}
		if brace_n == 0 {
			break
		}
	}
	toks = toks[first+1 : *i]
	parts, errs := Parts(toks, lex.ID_COMMA, true)
	b.Errors = append(b.Errors, errs...)
	return parts
}

func (b *Builder) idDataTypePartEnd(t *models.Type, dtv *strings.Builder, toks []lex.Token, i *int) {
	if *i+1 >= len(toks) {
		return
	}
	*i++
	tok := toks[*i]
	if tok.Id != lex.ID_BRACE || tok.Kind != lex.KND_LBRACKET {
		*i--
		return
	}
	dtv.WriteByte('[')
	var genericsStr strings.Builder
	parts := b.idGenericsParts(toks, i)
	generics := make([]models.Type, len(parts))
	for i, part := range parts {
		index := 0
		t, _ := b.DataType(part, &index, false, true)
		if index+1 < len(part) {
			b.pusherr(part[index+1], "invalid_syntax")
		}
		genericsStr.WriteString(t.String())
		genericsStr.WriteByte(',')
		generics[i] = t
	}
	dtv.WriteString(genericsStr.String()[:genericsStr.Len()-1])
	dtv.WriteByte(']')
	t.Tag = generics
}

func (b *Builder) datatype(t *models.Type, toks []lex.Token, i *int, arrays, err bool) (ok bool) {
	defer func() { t.Original = *t }()
	first := *i
	var dtv strings.Builder
	for ; *i < len(toks); *i++ {
		tok := toks[*i]
		switch tok.Id {
		case lex.ID_DT:
			t.Token = tok
			t.Id = juletype.TypeFromId(t.Token.Kind)
			dtv.WriteString(t.Token.Kind)
			ok = true
			goto ret
		case lex.ID_IDENT:
			dtv.WriteString(tok.Kind)
			if *i+1 < len(toks) && toks[*i+1].Id == lex.ID_DBLCOLON {
				break
			}
			t.Id = juletype.ID
			t.Token = tok
			b.idDataTypePartEnd(t, &dtv, toks, i)
			ok = true
			goto ret
		case lex.ID_CPP:
			if *i+1 >= len(toks) {
				if err {
					b.pusherr(tok, "invalid_syntax")
				}
				return
			}
			*i++
			if toks[*i].Id != lex.ID_DOT {
				if err {
					b.pusherr(toks[*i], "invalid_syntax")
				}
			}
			if *i+1 >= len(toks) {
				if err {
					b.pusherr(tok, "invalid_syntax")
				}
				return
			}
			*i++
			if toks[*i].Id != lex.ID_IDENT {
				if err {
					b.pusherr(toks[*i], "invalid_syntax")
				}
			}
			t.CppLinked = true
			t.Id = juletype.ID
			t.Token = toks[*i]
			dtv.WriteString(t.Token.Kind)
			b.idDataTypePartEnd(t, &dtv, toks, i)
			ok = true
			goto ret
		case lex.ID_DBLCOLON:
			dtv.WriteString(tok.Kind)
		case lex.ID_UNSAFE:
			if *i+1 >= len(toks) || toks[*i+1].Id != lex.ID_FN {
				t.Id = juletype.UNSAFE
				t.Token = tok
				dtv.WriteString(tok.Kind)
				ok = true
				goto ret
			}
			fallthrough
		case lex.ID_FN:
			t.Token = tok
			t.Id = juletype.FN
			f, proto_ok := b.funcPrototype(toks, i, false, true)
			if !proto_ok {
				b.pusherr(tok, "invalid_type")
				return false
			}
			*i--
			t.Tag = &f
			dtv.WriteString(f.DataTypeString())
			ok = true
			goto ret
		case lex.ID_OP:
			switch tok.Kind {
			case lex.KND_STAR, lex.KND_AMPER, lex.KND_DBL_AMPER:
				dtv.WriteString(tok.Kind)
			default:
				if err {
					b.pusherr(tok, "invalid_syntax")
				}
				return
			}
		case lex.ID_BRACE:
			switch tok.Kind {
			case lex.KND_LBRACKET:
				*i++
				if *i >= len(toks) {
					if err {
						b.pusherr(tok, "invalid_syntax")
					}
					return
				}
				tok = toks[*i]
				if tok.Id == lex.ID_BRACE && tok.Kind == lex.KND_RBRACKET {
					arrays = false
					dtv.WriteString(jule.PREFIX_SLICE)
					t.ComponentType = new(models.Type)
					t.Id = juletype.SLICE
					t.Token = tok
					*i++
					ok = b.datatype(t.ComponentType, toks, i, arrays, err)
					dtv.WriteString(t.ComponentType.Kind)
					goto ret
				}
				*i-- // Start from bracket
				if arrays {
					ok = b.MapOrArrayDataType(t, toks, i, err)
				} else {
					ok = b.MapDataType(t, toks, i, err)
				}
				if t.Id == juletype.VOID {
					if err {
						b.pusherr(tok, "invalid_syntax")
					}
					return
				}
				t.Token = tok
				t.Kind = dtv.String() + t.Kind
				return
			}
			return
		default:
			if err {
				b.pusherr(tok, "invalid_syntax")
			}
			return
		}
	}
	if err {
		b.pusherr(toks[first], "invalid_type")
	}
ret:
	t.Kind = dtv.String()
	return
}

// DataType builds AST model of data-type.
func (b *Builder) DataType(toks []lex.Token, i *int, arrays, err bool) (t models.Type, ok bool) {
	tok := toks[*i]
	ok = b.datatype(&t, toks, i, arrays, err)
	if err && t.Token.Id == lex.ID_NA {
		b.pusherr(tok, "invalid_type")
	}
	return
}

func (b *Builder) arrayDataType(t *models.Type, toks []lex.Token, i *int, err bool) (ok bool) {
	defer func() { t.Original = *t }()
	if *i+1 >= len(toks) {
		return
	}
	t.Id = juletype.ARRAY
	*i++
	exprI := *i
	t.ComponentType = new(models.Type)
	ok = b.datatype(t.ComponentType, toks, i, true, err)
	if !ok {
		return
	}
	if t.ComponentType.Size.AutoSized {
		b.pusherr(t.ComponentType.Size.Expr.Tokens[0], "invalid_syntax")
		ok = false
	}
	_, exprToks := RangeLast(toks[:exprI])
	exprToks = exprToks[1 : len(exprToks)-1]
	tok := exprToks[0]
	if len(exprToks) == 1 && tok.Id == lex.ID_OP && tok.Kind == lex.KND_TRIPLE_DOT {
		t.Size.AutoSized = true
		t.Size.Expr.Tokens = exprToks
	} else {
		t.Size.Expr = b.Expr(exprToks)
	}
	t.Kind = jule.PREFIX_ARRAY + t.ComponentType.Kind
	return
}

func (b *Builder) MapOrArrayDataType(t *models.Type, toks []lex.Token, i *int, err bool) (ok bool) {
	ok = b.MapDataType(t, toks, i, err)
	if !ok {
		ok = b.arrayDataType(t, toks, i, err)
	}
	return
}

// MapDataType builds map data-type.
func (b *Builder) MapDataType(t *models.Type, toks []lex.Token, i *int, err bool) (ok bool) {
	typeToks, colon := SplitColon(toks, i)
	if typeToks == nil || colon == -1 {
		return
	}
	return b.mapDataType(t, toks, typeToks, colon, err)
}

func (b *Builder) mapDataType(t *models.Type, toks, typeToks []lex.Token,
	colon int, err bool) (ok bool) {
	defer func() { t.Original = *t }()
	t.Id = juletype.MAP
	t.Token = toks[0]
	colonTok := toks[colon]
	if colon == 0 || colon+1 >= len(typeToks) {
		if err {
			b.pusherr(colonTok, "missing_expr")
		}
		return
	}
	keyTypeToks := typeToks[:colon]
	valueTypeToks := typeToks[colon+1:]
	types := make([]models.Type, 2)
	j := 0
	types[0], _ = b.DataType(keyTypeToks, &j, true, err)
	j = 0
	types[1], _ = b.DataType(valueTypeToks, &j, true, err)
	t.Tag = types
	t.Kind = t.MapKind()
	ok = true
	return
}

func (b *Builder) funcMultiTypeRet(toks []lex.Token, i *int) (t models.RetType, ok bool) {
	tok := toks[*i]
	t.Type.Kind += tok.Kind
	*i++
	if *i >= len(toks) {
		*i--
		t.Type, ok = b.DataType(toks, i, false, false)
		return
	}
	tok = toks[*i]
	*i-- // For point to parenthses - ( -
	rang := Range(i, lex.KND_LPAREN, lex.KND_RPARENT, toks)
	params := b.Params(rang, false, true)
	types := make([]models.Type, len(params))
	for i, param := range params {
		types[i] = param.Type
		if param.Id != jule.ANONYMOUS {
			param.Token.Kind = param.Id
		} else {
			param.Token.Kind = juleapi.IGNORE
		}
		t.Identifiers = append(t.Identifiers, param.Token)
	}
	if len(types) > 1 {
		t.Type.MultiTyped = true
		t.Type.Tag = types
	} else {
		t.Type = types[0]
	}
	// Decrament for correct block parsing
	*i--
	ok = true
	return
}

// FuncRetDataType builds ret data-type of function.
func (b *Builder) FuncRetDataType(toks []lex.Token, i *int) (t models.RetType, ok bool) {
	t.Type.Id = juletype.VOID
	t.Type.Kind = juletype.TYPE_MAP[t.Type.Id]
	if *i >= len(toks) {
		return
	}
	tok := toks[*i]
	switch tok.Id {
	case lex.ID_BRACE:
		switch tok.Kind {
		case lex.KND_LPAREN:
			return b.funcMultiTypeRet(toks, i)
		case lex.KND_LBRACE:
			return
		}
	case lex.ID_OP:
		if tok.Kind == lex.KND_EQ {
			return
		}
	}
	t.Type, ok = b.DataType(toks, i, false, true)
	return
}

func (b *Builder) pushStatementToBlock(bs *blockStatement) {
	if len(bs.toks) == 0 {
		return
	}
	lastTok := bs.toks[len(bs.toks)-1]
	if lastTok.Id == lex.ID_SEMICOLON {
		if len(bs.toks) == 1 {
			return
		}
		bs.toks = bs.toks[:len(bs.toks)-1]
	}
	s := b.Statement(bs)
	if s.Data == nil {
		return
	}
	s.WithTerminator = bs.withTerminator
	bs.block.Tree = append(bs.block.Tree, s)
}

func setToNextStatement(bs *blockStatement) {
	*bs.srcToks = (*bs.srcToks)[bs.pos:]
	bs.pos, bs.withTerminator = NextStatementPos(*bs.srcToks, 0)
	if bs.withTerminator {
		bs.toks = (*bs.srcToks)[:bs.pos-1]
	} else {
		bs.toks = (*bs.srcToks)[:bs.pos]
	}
}

func blockStatementFinished(bs *blockStatement) bool {
	return bs.pos >= len(*bs.srcToks)
}

// Block builds AST model of statements of code block.
func (b *Builder) Block(toks []lex.Token) (block *models.Block) {
	block = new(models.Block)
	var bs blockStatement
	bs.block = block
	bs.srcToks = &toks
	for {
		setToNextStatement(&bs)
		b.pushStatementToBlock(&bs)
	next:
		if len(bs.nextToks) > 0 {
			bs.toks = bs.nextToks
			bs.nextToks = nil
			b.pushStatementToBlock(&bs)
			goto next
		}
		if blockStatementFinished(&bs) {
			break
		}
	}
	return
}

// Statement builds AST model of statement.
func (b *Builder) Statement(bs *blockStatement) (s models.Statement) {
	tok := bs.toks[0]
	if tok.Id == lex.ID_IDENT {
		s, ok := b.IdStatement(bs)
		if ok {
			return s
		}
	}
	s, ok := b.AssignStatement(bs.toks)
	if ok {
		return s
	}
	switch tok.Id {
	case lex.ID_CONST, lex.ID_LET, lex.ID_MUT:
		return b.VarStatement(bs, true)
	case lex.ID_RET:
		return b.RetStatement(bs.toks)
	case lex.ID_ITER:
		return b.IterExpr(bs)
	case lex.ID_BREAK:
		return b.BreakStatement(bs.toks)
	case lex.ID_CONTINUE:
		return b.ContinueStatement(bs.toks)
	case lex.ID_IF:
		return b.IfExpr(bs)
	case lex.ID_ELSE:
		return b.ElseBlock(bs)
	case lex.ID_COMMENT:
		return b.CommentStatement(bs.toks[0])
	case lex.ID_CO:
		return b.ConcurrentCallStatement(bs.toks)
	case lex.ID_GOTO:
		return b.GotoStatement(bs.toks)
	case lex.ID_FALLTHROUGH:
		return b.Fallthrough(bs.toks)
	case lex.ID_TYPE:
		t := b.TypeAlias(bs.toks)
		s.Token = t.Token
		s.Data = t
		return
	case lex.ID_MATCH:
		return b.MatchCase(bs.toks)
	case lex.ID_UNSAFE:
		if len(bs.toks) == 1 || bs.toks[1].Kind != lex.KND_LBRACE {
			break
		}
		return b.blockStatement(bs.toks[1:], true)
	case lex.ID_BRACE:
		if tok.Kind == lex.KND_LBRACE {
			return b.blockStatement(bs.toks, false)
		}
	}
	if IsFuncCall(bs.toks) != nil {
		return b.ExprStatement(bs)
	}
	b.pusherr(tok, "invalid_syntax")
	return
}

func (b *Builder) blockStatement(toks []lex.Token, is_unsafe bool) models.Statement {
	i := 0
	tok := toks[0]
	toks = Range(&i, lex.KND_LBRACE, lex.KND_RBRACE, toks)
	if i < len(toks) {
		b.pusherr(toks[i], "invalid_syntax")
	}
	block := b.Block(toks)
	block.IsUnsafe = is_unsafe
	return models.Statement{Token: tok, Data: block}
}

func (b *Builder) assignInfo(toks []lex.Token) (info AssignInfo) {
	info.Ok = true
	brace_n := 0
	for i, tok := range toks {
		if tok.Id == lex.ID_BRACE {
			switch tok.Kind {
			case lex.KND_LBRACE, lex.KND_LBRACKET, lex.KND_LPAREN:
				brace_n++
			default:
				brace_n--
			}
		}
		if brace_n > 0 {
			continue
		} else if tok.Id != lex.ID_OP {
			continue
		} else if !IsAssignOperator(tok.Kind) {
			continue
		}
		info.Left = toks[:i]
		if info.Left == nil {
			b.pusherr(tok, "invalid_syntax")
			info.Ok = false
		}
		info.Setter = tok
		if i+1 >= len(toks) {
			info.Right = nil
			info.Ok = IsPostfixOperator(info.Setter.Kind)
			break
		}
		info.Right = toks[i+1:]
		if IsPostfixOperator(info.Setter.Kind) {
			if info.Right != nil {
				b.pusherr(info.Right[0], "invalid_syntax")
				info.Right = nil
			}
		}
		break
	}
	return
}

func (b *Builder) buildAssignLeft(toks []lex.Token) (left models.AssignLeft) {
	left.Expr.Tokens = toks
	if left.Expr.Tokens[0].Id == lex.ID_IDENT {
		left.Var.Token = left.Expr.Tokens[0]
		left.Var.Id = left.Var.Token.Kind
	}
	left.Expr = b.Expr(left.Expr.Tokens)
	return
}

func (b *Builder) assignLefts(parts [][]lex.Token) []models.AssignLeft {
	var lefts []models.AssignLeft
	for _, part := range parts {
		left := b.buildAssignLeft(part)
		lefts = append(lefts, left)
	}
	return lefts
}

func (b *Builder) assignExprs(toks []lex.Token) []models.Expr {
	parts, errs := Parts(toks, lex.ID_COMMA, true)
	if len(errs) > 0 {
		b.Errors = append(b.Errors, errs...)
		return nil
	}
	exprs := make([]models.Expr, len(parts))
	for i, part := range parts {
		exprs[i] = b.Expr(part)
	}
	return exprs
}

// AssignStatement builds AST model of assignment statement.
func (b *Builder) AssignStatement(toks []lex.Token) (s models.Statement, _ bool) {
	assign, ok := b.AssignExpr(toks)
	if !ok {
		return
	}
	s.Token = toks[0]
	s.Data = assign
	return s, true
}

// AssignExpr builds AST model of assignment expression.
func (b *Builder) AssignExpr(toks []lex.Token) (assign models.Assign, ok bool) {
	if !CheckAssignTokens(toks) {
		return
	}
	switch toks[0].Id {
	case lex.ID_LET:
		return b.letDeclAssign(toks)
	default:
		return b.plainAssign(toks)
	}
}

func (b *Builder) letDeclAssign(toks []lex.Token) (assign models.Assign, ok bool) {
	if len(toks) < 1 {
		return
	}
	// Skip "let" keyword
	toks = toks[1:]
	tok := toks[0]
	if tok.Id != lex.ID_BRACE || tok.Kind != lex.KND_LPAREN {
		return
	}
	ok = true
	var i int
	rang := Range(&i, lex.KND_LPAREN, lex.KND_RPARENT, toks)
	if rang == nil {
		b.pusherr(tok, "invalid_syntax")
		return
	} else if i+1 < len(toks) {
		assign.Setter = toks[i]
		i++
		assign.Right = b.assignExprs(toks[i:])
	}
	parts, errs := Parts(rang, lex.ID_COMMA, true)
	if len(errs) > 0 {
		b.Errors = append(b.Errors, errs...)
		return
	}
	for _, part := range parts {
		if len(part) > 2 {
			b.pusherr(part[2], "invalid_syntax")
		}
		mutable := false
		tok := part[0]
		if tok.Id == lex.ID_MUT {
			mutable = true
			part = part[1:]
			if len(part) == 0 {
				b.pusherr(tok, "invalid_syntax")
				continue
			}
		}
		left := b.buildAssignLeft(part)
		left.Var.Mutable = mutable
		left.Var.New = !juleapi.IsIgnoreId(left.Var.Id)
		left.Var.SetterTok = assign.Setter
		assign.Left = append(assign.Left, left)
	}
	return
}

func (b *Builder) plainAssign(toks []lex.Token) (assign models.Assign, ok bool) {
	info := b.assignInfo(toks)
	if !info.Ok {
		return
	}
	ok = true
	assign.Setter = info.Setter
	parts, errs := Parts(info.Left, lex.ID_COMMA, true)
	if len(errs) > 0 {
		b.Errors = append(b.Errors, errs...)
		return
	}
	assign.Left = b.assignLefts(parts)
	if info.Right != nil {
		assign.Right = b.assignExprs(info.Right)
	}
	return
}

// BuildReturnStatement builds AST model of return statement.
func (b *Builder) IdStatement(bs *blockStatement) (s models.Statement, ok bool) {
	if len(bs.toks) == 1 {
		return
	}
	tok := bs.toks[1]
	switch tok.Id {
	case lex.ID_COLON:
		return b.LabelStatement(bs), true
	}
	return
}

// LabelStatement builds AST model of label.
func (b *Builder) LabelStatement(bs *blockStatement) models.Statement {
	var l models.Label
	l.Token = bs.toks[0]
	l.Label = l.Token.Kind
	if len(bs.toks) > 2 {
		bs.nextToks = bs.toks[2:]
	}
	return models.Statement{Token: l.Token, Data: l}
}

// ExprStatement builds AST model of expression.
func (b *Builder) ExprStatement(bs *blockStatement) models.Statement {
	expr := models.ExprStatement{
		Expr: b.Expr(bs.toks),
	}
	return models.Statement{
		Token: bs.toks[0],
		Data:  expr,
	}
}

// Args builds AST model of arguments.
func (b *Builder) Args(toks []lex.Token, targeting bool) *models.Args {
	args := new(models.Args)
	last := 0
	brace_n := 0
	for i, tok := range toks {
		if tok.Id == lex.ID_BRACE {
			switch tok.Kind {
			case lex.KND_LBRACE, lex.KND_LBRACKET, lex.KND_LPAREN:
				brace_n++
			default:
				brace_n--
			}
		}
		if brace_n > 0 || tok.Id != lex.ID_COMMA {
			continue
		}
		b.pushArg(args, targeting, toks[last:i], tok)
		last = i + 1
	}
	if last < len(toks) {
		if last == 0 {
			if len(toks) > 0 {
				b.pushArg(args, targeting, toks[last:], toks[last])
			}
		} else {
			b.pushArg(args, targeting, toks[last:], toks[last-1])
		}
	}
	return args
}

func (b *Builder) pushArg(args *models.Args, targeting bool, toks []lex.Token, err lex.Token) {
	if len(toks) == 0 {
		b.pusherr(err, "invalid_syntax")
		return
	}
	var arg models.Arg
	arg.Token = toks[0]
	if targeting && arg.Token.Id == lex.ID_IDENT {
		if len(toks) > 1 {
			tok := toks[1]
			if tok.Id == lex.ID_COLON {
				args.Targeted = true
				arg.TargetId = arg.Token.Kind
				toks = toks[2:]
			}
		}
	}
	arg.Expr = b.Expr(toks)
	args.Src = append(args.Src, arg)
}

func (b *Builder) varBegin(v *models.Var, i *int, toks []lex.Token) {
	tok := toks[*i]
	switch tok.Id {
	case lex.ID_LET:
		// Initialize 1 for skip the let keyword
		*i++
		if toks[*i].Id == lex.ID_MUT {
			v.Mutable = true
			// Skip the mut keyword
			*i++
		}
	case lex.ID_CONST:
		*i++
		if v.Const {
			b.pusherr(tok, "already_const")
			break
		}
		v.Const = true
		if !v.Mutable {
			break
		}
		fallthrough
	default:
		b.pusherr(tok, "invalid_syntax")
		return
	}
	if *i >= len(toks) {
		b.pusherr(tok, "invalid_syntax")
	}
}

func (b *Builder) varTypeNExpr(v *models.Var, toks []lex.Token, i int, expr bool) {
	tok := toks[i]
	if tok.Id == lex.ID_COLON {
		i++ // Skip type annotation operator (:)
		if i >= len(toks) ||
			(toks[i].Id == lex.ID_OP && toks[i].Kind == lex.KND_EQ) {
			b.pusherr(tok, "missing_type")
			return
		}
		t, ok := b.DataType(toks, &i, true, false)
		if ok {
			v.Type = t
			i++
			if i >= len(toks) {
				return
			}
			tok = toks[i]
		}
	}
	if expr && tok.Id == lex.ID_OP {
		if tok.Kind != lex.KND_EQ {
			b.pusherr(tok, "invalid_syntax")
			return
		}
		valueToks := toks[i+1:]
		if len(valueToks) == 0 {
			b.pusherr(tok, "missing_expr")
			return
		}
		v.Expr = b.Expr(valueToks)
		v.SetterTok = tok
	} else {
		b.pusherr(tok, "invalid_syntax")
	}
}

// Var builds AST model of variable statement.
func (b *Builder) Var(toks []lex.Token, begin, expr bool) (v models.Var) {
	v.Pub = b.pub
	b.pub = false
	i := 0
	v.Token = toks[i]
	if begin {
		b.varBegin(&v, &i, toks)
		if i >= len(toks) {
			return
		}
	}
	v.Token = toks[i]
	if v.Token.Id != lex.ID_IDENT {
		b.pusherr(v.Token, "invalid_syntax")
		return
	}
	v.Id = v.Token.Kind
	v.Type.Id = juletype.VOID
	v.Type.Kind = juletype.TYPE_MAP[v.Type.Id]
	if i >= len(toks) {
		return
	}
	i++
	if i < len(toks) {
		b.varTypeNExpr(&v, toks, i, expr)
	} else if !expr {
		b.pusherr(v.Token, "missing_type")
	}
	return
}

// VarStatement builds AST model of variable declaration statement.
func (b *Builder) VarStatement(bs *blockStatement, expr bool) models.Statement {
	v := b.Var(bs.toks, true, expr)
	v.Owner = bs.block
	return models.Statement{Token: v.Token, Data: v}
}

// CommentStatement builds AST model of comment statement.
func (b *Builder) CommentStatement(tok lex.Token) (s models.Statement) {
	s.Token = tok
	tok.Kind = strings.TrimSpace(tok.Kind[2:])
	s.Data = models.Comment{Content: tok.Kind}
	return
}

func (b *Builder) ConcurrentCallStatement(toks []lex.Token) (s models.Statement) {
	var cc models.ConcurrentCall
	cc.Token = toks[0]
	toks = toks[1:]
	if len(toks) == 0 {
		b.pusherr(cc.Token, "missing_expr")
		return
	}
	if IsFuncCall(toks) == nil {
		b.pusherr(cc.Token, "expr_not_func_call")
	}
	cc.Expr = b.Expr(toks)
	s.Token = cc.Token
	s.Data = cc
	return
}

func (b *Builder) Fallthrough(toks []lex.Token) (s models.Statement) {
	s.Token = toks[0]
	if len(toks) > 1 {
		b.pusherr(toks[1], "invalid_syntax")
	}
	s.Data = models.Fallthrough{
		Token: s.Token,
	}
	return
}

func (b *Builder) GotoStatement(toks []lex.Token) (s models.Statement) {
	s.Token = toks[0]
	if len(toks) == 1 {
		b.pusherr(s.Token, "missing_goto_label")
		return
	} else if len(toks) > 2 {
		b.pusherr(toks[2], "invalid_syntax")
	}
	idTok := toks[1]
	if idTok.Id != lex.ID_IDENT {
		b.pusherr(idTok, "invalid_syntax")
		return
	}
	var gt models.Goto
	gt.Token = s.Token
	gt.Label = idTok.Kind
	s.Data = gt
	return
}

// RetStatement builds AST model of return statement.
func (b *Builder) RetStatement(toks []lex.Token) models.Statement {
	var ret models.Ret
	ret.Token = toks[0]
	if len(toks) > 1 {
		ret.Expr = b.Expr(toks[1:])
	}
	return models.Statement{
		Token: ret.Token,
		Data:  ret,
	}
}

func (b *Builder) getWhileIterProfile(toks []lex.Token) models.IterWhile {
	return models.IterWhile{
		Expr: b.Expr(toks),
	}
}

func (b *Builder) getForeachVarsToks(toks []lex.Token) [][]lex.Token {
	vars, errs := Parts(toks, lex.ID_COMMA, true)
	b.Errors = append(b.Errors, errs...)
	return vars
}

func (b *Builder) getVarProfile(toks []lex.Token) (v models.Var) {
	if len(toks) == 0 {
		return
	}
	v.Token = toks[0]
	if v.Token.Id == lex.ID_MUT {
		v.Mutable = true
		if len(toks) == 1 {
			b.pusherr(v.Token, "invalid_syntax")
		}
		v.Token = toks[1]
	} else if len(toks) > 1 {
		b.pusherr(toks[1], "invalid_syntax")
	}
	if v.Token.Id != lex.ID_IDENT {
		b.pusherr(v.Token, "invalid_syntax")
		return
	}
	v.Id = v.Token.Kind
	v.New = true
	return
}

func (b *Builder) getForeachIterVars(varsToks [][]lex.Token) []models.Var {
	var vars []models.Var
	for _, toks := range varsToks {
		vars = append(vars, b.getVarProfile(toks))
	}
	return vars
}

func (b *Builder) setup_foreach_explicit_vars(f *models.IterForeach, toks []lex.Token) {
	i := 0
	rang := Range(&i, lex.KND_LPAREN, lex.KND_RPARENT, toks)
	if i < len(toks) {
		b.pusherr(f.InToken, "invalid_syntax")
	}
	b.setup_foreach_plain_vars(f, rang)
}

func (b *Builder) setup_foreach_plain_vars(f *models.IterForeach, toks []lex.Token) {
	varsToks := b.getForeachVarsToks(toks)
	if len(varsToks) == 0 {
		return
	}
	if len(varsToks) > 2 {
		b.pusherr(f.InToken, "much_foreach_vars")
	}
	vars := b.getForeachIterVars(varsToks)
	f.KeyA = vars[0]
	if len(vars) > 1 {
		f.KeyB = vars[1]
	} else {
		f.KeyB.Id = juleapi.IGNORE
	}
}

func (b *Builder) setup_foreach_vars(f *models.IterForeach, toks []lex.Token) {
	if toks[0].Id == lex.ID_BRACE {
		if toks[0].Kind != lex.KND_LPAREN {
			b.pusherr(toks[0], "invalid_syntax")
			return
		}
		b.setup_foreach_explicit_vars(f, toks)
		return
	}
	b.setup_foreach_plain_vars(f, toks)
}

func (b *Builder) getForeachIterProfile(varToks, exprToks []lex.Token, inTok lex.Token) models.IterForeach {
	var foreach models.IterForeach
	foreach.InToken = inTok
	if len(exprToks) == 0 {
		b.pusherr(inTok, "missing_expr")
		return foreach
	}
	foreach.Expr = b.Expr(exprToks)
	if len(varToks) == 0 {
		foreach.KeyA.Id = juleapi.IGNORE
		foreach.KeyB.Id = juleapi.IGNORE
	} else {
		b.setup_foreach_vars(&foreach, varToks)
	}
	return foreach
}

func (b *Builder) getIterProfile(toks []lex.Token, errtok lex.Token) models.IterProfile {
	brace_n := 0
	for i, tok := range toks {
		if tok.Id == lex.ID_BRACE {
			switch tok.Kind {
			case lex.KND_LBRACE, lex.KND_LBRACKET, lex.KND_LPAREN:
				brace_n++
				continue
			default:
				brace_n--
			}
		}
		if brace_n != 0 {
			continue
		}
		switch tok.Id {
		case lex.ID_IN:
			varToks := toks[:i]
			exprToks := toks[i+1:]
			return b.getForeachIterProfile(varToks, exprToks, tok)
		}
	}
	return b.getWhileIterProfile(toks)
}

func (b *Builder) forStatement(toks []lex.Token) models.Statement {
	s := b.Statement(&blockStatement{toks: toks})
	switch s.Data.(type) {
	case models.ExprStatement, models.Assign, models.Var:
	default:
		b.pusherr(toks[0], "invalid_syntax")
	}
	return s
}

func (b *Builder) forIterProfile(bs *blockStatement) (s models.Statement) {
	var iter models.Iter
	iter.Token = bs.toks[0]
	bs.toks = bs.toks[1:]
	var profile models.IterFor
	if len(bs.toks) > 0 {
		profile.Once = b.forStatement(bs.toks)
	}
	if blockStatementFinished(bs) {
		b.pusherr(iter.Token, "invalid_syntax")
		return
	}
	setToNextStatement(bs)
	if len(bs.toks) > 0 {
		profile.Condition = b.Expr(bs.toks)
	}
	if blockStatementFinished(bs) {
		b.pusherr(iter.Token, "invalid_syntax")
		return
	}
	setToNextStatement(bs)
	exprToks := BlockExpr(bs.toks)
	if len(exprToks) > 0 {
		profile.Next = b.forStatement(exprToks)
	}
	i := len(exprToks)
	blockToks := b.getrange(&i, lex.KND_LBRACE, lex.KND_RBRACE, &bs.toks)
	if blockToks == nil {
		b.pusherr(iter.Token, "body_not_exist")
		return
	}
	if i < len(bs.toks) {
		b.pusherr(bs.toks[i], "invalid_syntax")
	}
	iter.Block = b.Block(blockToks)
	iter.Profile = profile
	return models.Statement{Token: iter.Token, Data: iter}
}

func (b *Builder) commonIterProfile(toks []lex.Token) (s models.Statement) {
	var iter models.Iter
	iter.Token = toks[0]
	toks = toks[1:]
	if len(toks) == 0 {
		b.pusherr(iter.Token, "body_not_exist")
		return
	}
	exprToks := BlockExpr(toks)
	if len(exprToks) > 0 {
		iter.Profile = b.getIterProfile(exprToks, iter.Token)
	}
	i := len(exprToks)
	blockToks := b.getrange(&i, lex.KND_LBRACE, lex.KND_RBRACE, &toks)
	if blockToks == nil {
		b.pusherr(iter.Token, "body_not_exist")
		return
	}
	if i < len(toks) {
		b.pusherr(toks[i], "invalid_syntax")
	}
	iter.Block = b.Block(blockToks)
	return models.Statement{Token: iter.Token, Data: iter}
}

func (b *Builder) IterExpr(bs *blockStatement) models.Statement {
	if bs.withTerminator {
		return b.forIterProfile(bs)
	}
	return b.commonIterProfile(bs.toks)
}

func (b *Builder) caseexprs(toks *[]lex.Token, caseIsDefault bool) []models.Expr {
	var exprs []models.Expr
	pushExpr := func(toks []lex.Token, tok lex.Token) {
		if caseIsDefault {
			if len(toks) > 0 {
				b.pusherr(tok, "invalid_syntax")
			}
			return
		}
		if len(toks) > 0 {
			exprs = append(exprs, b.Expr(toks))
			return
		}
		b.pusherr(tok, "missing_expr")
	}
	brace_n := 0
	j := 0
	var i int
	var tok lex.Token
	for i, tok = range *toks {
		if tok.Id == lex.ID_BRACE {
			switch tok.Kind {
			case lex.KND_LPAREN, lex.KND_LBRACE, lex.KND_LBRACKET:
				brace_n++
			default:
				brace_n--
			}
			continue
		} else if brace_n != 0 {
			continue
		}
		switch tok.Id {
		case lex.ID_COMMA:
			pushExpr((*toks)[j:i], tok)
			j = i + 1
		case lex.ID_COLON:
			pushExpr((*toks)[j:i], tok)
			*toks = (*toks)[i+1:]
			return exprs
		}
	}
	b.pusherr((*toks)[0], "invalid_syntax")
	*toks = nil
	return nil
}

func (b *Builder) caseblock(toks *[]lex.Token) *models.Block {
	brace_n := 0
	for i, tok := range *toks {
		if tok.Id == lex.ID_BRACE {
			switch tok.Kind {
			case lex.KND_LPAREN, lex.KND_LBRACE, lex.KND_LBRACKET:
				brace_n++
			default:
				brace_n--
			}
			continue
		} else if brace_n != 0 {
			continue
		}
		switch tok.Id {
		case lex.ID_CASE, lex.ID_DEFAULT:
			blockToks := (*toks)[:i]
			*toks = (*toks)[i:]
			return b.Block(blockToks)
		}
	}
	block := b.Block(*toks)
	*toks = nil
	return block
}

func (b *Builder) getcase(toks *[]lex.Token) models.Case {
	var c models.Case
	c.Token = (*toks)[0]
	*toks = (*toks)[1:]
	c.Exprs = b.caseexprs(toks, c.Token.Id == lex.ID_DEFAULT)
	c.Block = b.caseblock(toks)
	return c
}

func (b *Builder) cases(toks []lex.Token) ([]models.Case, *models.Case) {
	var cases []models.Case
	var def *models.Case
	for len(toks) > 0 {
		tok := toks[0]
		switch tok.Id {
		case lex.ID_CASE:
			cases = append(cases, b.getcase(&toks))
		case lex.ID_DEFAULT:
			c := b.getcase(&toks)
			c.Token = tok
			if def == nil {
				def = new(models.Case)
				*def = c
				break
			}
			fallthrough
		default:
			b.pusherr(tok, "invalid_syntax")
		}
	}
	return cases, def
}

// MatchCase builds AST model of match-case.
func (b *Builder) MatchCase(toks []lex.Token) (s models.Statement) {
	match := new(models.Match)
	match.Token = toks[0]
	s.Token = match.Token
	toks = toks[1:]
	exprToks := BlockExpr(toks)
	if len(exprToks) > 0 {
		match.Expr = b.Expr(exprToks)
	}
	i := len(exprToks)
	blockToks := b.getrange(&i, lex.KND_LBRACE, lex.KND_RBRACE, &toks)
	if blockToks == nil {
		b.pusherr(match.Token, "body_not_exist")
		return
	}
	match.Cases, match.Default = b.cases(blockToks)
	for i := range match.Cases {
		c := &match.Cases[i]
		c.Match = match
		if i > 0 {
			match.Cases[i-1].Next = c
		}
	}
	if match.Default != nil {
		if len(match.Cases) > 0 {
			match.Cases[len(match.Cases)-1].Next = match.Default
		}
		match.Default.Match = match
	}
	s.Data = match
	return
}

// IfExpr builds AST model of if expression.
func (b *Builder) IfExpr(bs *blockStatement) (s models.Statement) {
	var ifast models.If
	ifast.Token = bs.toks[0]
	bs.toks = bs.toks[1:]
	exprToks := BlockExpr(bs.toks)
	i := 0
	if len(exprToks) == 0 {
		if len(bs.toks) == 0 || bs.pos >= len(*bs.srcToks) {
			b.pusherr(ifast.Token, "missing_expr")
			return
		}
		exprToks = bs.toks
		setToNextStatement(bs)
	} else {
		i = len(exprToks)
	}
	blockToks := b.getrange(&i, lex.KND_LBRACE, lex.KND_RBRACE, &bs.toks)
	if blockToks == nil {
		b.pusherr(ifast.Token, "body_not_exist")
		return
	}
	if i < len(bs.toks) {
		if bs.toks[i].Id == lex.ID_ELSE {
			bs.nextToks = bs.toks[i:]
		} else {
			b.pusherr(bs.toks[i], "invalid_syntax")
		}
	}
	ifast.Expr = b.Expr(exprToks)
	ifast.Block = b.Block(blockToks)
	return models.Statement{Token: ifast.Token, Data: ifast}
}

// ElseIfEpxr builds AST model of else if expression.
func (b *Builder) ElseIfExpr(bs *blockStatement) (s models.Statement) {
	var elif models.ElseIf
	elif.Token = bs.toks[1]
	bs.toks = bs.toks[2:]
	exprToks := BlockExpr(bs.toks)
	i := 0
	if len(exprToks) == 0 {
		if len(bs.toks) == 0 || bs.pos >= len(*bs.srcToks) {
			b.pusherr(elif.Token, "missing_expr")
			return
		}
		exprToks = bs.toks
		setToNextStatement(bs)
	} else {
		i = len(exprToks)
	}
	blockToks := b.getrange(&i, lex.KND_LBRACE, lex.KND_RBRACE, &bs.toks)
	if blockToks == nil {
		b.pusherr(elif.Token, "body_not_exist")
		return
	}
	if i < len(bs.toks) {
		if bs.toks[i].Id == lex.ID_ELSE {
			bs.nextToks = bs.toks[i:]
		} else {
			b.pusherr(bs.toks[i], "invalid_syntax")
		}
	}
	elif.Expr = b.Expr(exprToks)
	elif.Block = b.Block(blockToks)
	return models.Statement{Token: elif.Token, Data: elif}
}

// ElseBlock builds AST model of else block.
func (b *Builder) ElseBlock(bs *blockStatement) (s models.Statement) {
	if len(bs.toks) > 1 && bs.toks[1].Id == lex.ID_IF {
		return b.ElseIfExpr(bs)
	}
	var elseast models.Else
	elseast.Token = bs.toks[0]
	bs.toks = bs.toks[1:]
	i := 0
	blockToks := b.getrange(&i, lex.KND_LBRACE, lex.KND_RBRACE, &bs.toks)
	if blockToks == nil {
		if i < len(bs.toks) {
			b.pusherr(elseast.Token, "else_have_expr")
		} else {
			b.pusherr(elseast.Token, "body_not_exist")
		}
		return
	}
	if i < len(bs.toks) {
		b.pusherr(bs.toks[i], "invalid_syntax")
	}
	elseast.Block = b.Block(blockToks)
	return models.Statement{Token: elseast.Token, Data: elseast}
}

// BreakStatement builds AST model of break statement.
func (b *Builder) BreakStatement(toks []lex.Token) models.Statement {
	var breakAST models.Break
	breakAST.Token = toks[0]
	if len(toks) > 1 {
		if toks[1].Id != lex.ID_IDENT {
			b.pusherr(toks[1], "invalid_syntax")
		} else {
			breakAST.LabelToken = toks[1]
			if len(toks) > 2 {
				b.pusherr(toks[1], "invalid_syntax")
			}
		}
	}
	return models.Statement{
		Token: breakAST.Token,
		Data:  breakAST,
	}
}

// ContinueStatement builds AST model of continue statement.
func (b *Builder) ContinueStatement(toks []lex.Token) models.Statement {
	var continueAST models.Continue
	continueAST.Token = toks[0]
	if len(toks) > 1 {
		if toks[1].Id != lex.ID_IDENT {
			b.pusherr(toks[1], "invalid_syntax")
		} else {
			continueAST.LoopLabel = toks[1]
			if len(toks) > 2 {
				b.pusherr(toks[1], "invalid_syntax")
			}
		}
	}
	return models.Statement{Token: continueAST.Token, Data:  continueAST}
}

// Expr builds AST model of expression.
func (b *Builder) Expr(toks []lex.Token) (e models.Expr) {
	e.Op = b.build_expr_op(toks)
	e.Tokens = toks
	return
}

func (b *Builder) build_binop_expr(toks []lex.Token) any {
	i := b.find_lowest_precedenced_operator(toks)
	if i != -1 {
		return b.build_binop(toks)
	}
	return models.BinopExpr{Tokens: toks}
}

func (b *Builder) build_binop(toks []lex.Token) models.Binop {
	op := models.Binop{}
	i := b.find_lowest_precedenced_operator(toks)
	op.L = b.build_binop_expr(toks[:i])
	op.R = b.build_binop_expr(toks[i+1:])
	op.Op = toks[i]
	return op
}

func eliminate_comments(toks []lex.Token) []lex.Token {
	cutted := []lex.Token{}
	for _, token := range toks {
		if token.Id != lex.ID_COMMENT {
			cutted = append(cutted, token)
		}
	}
	return cutted
}

// Returns BinopExpr or Binop instance for expression Op.
func (b *Builder) build_expr_op(toks []lex.Token) any {
	toks = eliminate_comments(toks)
	i := b.find_lowest_precedenced_operator(toks)
	if i == -1 {
		return b.build_binop_expr(toks)
	}
	return b.build_binop(toks)
}

// nextOperator find index of priority operator and returns index of operator
// if found, returns -1 if not.
func (b *Builder) find_lowest_precedenced_operator(toks []lex.Token) int {
	prec := precedencer{}
	brace_n := 0
	for i, token := range toks {
		switch {
		case token.Id == lex.ID_BRACE:
			switch token.Kind {
			case lex.KND_LBRACE, lex.KND_LPAREN, lex.KND_LBRACKET:
				brace_n++
			default:
				brace_n--
			}
			continue
		case i == 0:
			continue
		case token.Id != lex.ID_OP:
			continue
		case brace_n > 0:
			continue
		}
		// Skip unary operator.
		if toks[i-1].Id == lex.ID_OP {
			continue
		}
		switch token.Kind {
		case lex.KND_STAR, lex.KND_PERCENT, lex.KND_SOLIDUS,
			lex.KND_RSHIFT, lex.KND_LSHIFT, lex.KND_AMPER:
			prec.set(5, i)
		case lex.KND_PLUS, lex.KND_MINUS, lex.KND_VLINE, lex.KND_CARET:
			prec.set(4, i)
		case lex.KND_EQS, lex.KND_NOT_EQ, lex.KND_LT,
			lex.KND_LESS_EQ, lex.KND_GT, lex.KND_GREAT_EQ:
			prec.set(3, i)
		case lex.KND_DBL_AMPER:
			prec.set(2, i)
		case lex.KND_DBL_VLINE:
			prec.set(1, i)
		}
	}
	data := prec.get_lower()
	if data == nil {
		return -1
	}
	return data.(int)
}

func (b *Builder) getrange(i *int, open, close string, toks *[]lex.Token) []lex.Token {
	rang := Range(i, open, close, *toks)
	if rang != nil {
		return rang
	}
	if b.Ended() {
		return nil
	}
	*i = 0
	*toks = b.nextBuilderStatement()
	rang = Range(i, open, close, *toks)
	return rang
}

func (b *Builder) skipStatement(i *int, toks *[]lex.Token) []lex.Token {
	start := *i
	*i, _ = NextStatementPos(*toks, start)
	stoks := (*toks)[start:*i]
	if stoks[len(stoks)-1].Id == lex.ID_SEMICOLON {
		if len(stoks) == 1 {
			return b.skipStatement(i, toks)
		}
		stoks = stoks[:len(stoks)-1]
	}
	return stoks
}

func (b *Builder) nextBuilderStatement() []lex.Token {
	return b.skipStatement(&b.Pos, &b.Tokens)
}
