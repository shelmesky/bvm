%{
package parser

func setResult(l yyLexer, v *Node) {
  l.(*lexer).result = v
}

%}

%union {
    n       *Node
    b       bool
    i       int64
    f       float64
    s       string
    sa      []string
    va      []NVar
}

// Identifiers + literals
%token<s> IDENT  // foobar
%token<s> ENV   // $foobar
%token<s> CALL  // foobar(
%token<s> CALLCONTRACT  // @foobar(
%token<s> INDEX  // foobar[
%token<i> INT    // 314
%token<f> FLOAT    // 3.14
%token<s> STRING  // "string"
%token<s> QSTRING  // `string`
%token<b> TRUE   // true
%token<b> FALSE  // false

// Delimiters
%token NEWLINE // \n
%token COMMA   // ,
%token COLON   // :
%token LPAREN  // (
%token RPAREN  // )
%token OBJ  // @{
%token LBRACE  // {
%token RBRACE  // }
%token LBRACKET // [
%token RBRACKET // ]
%token QUESTION // ?
%token DOUBLEDOT   // ..
%token DOT   // .

// Operators
%token ADD // +
%token SUB // -
%token MUL // *
%token DIV // /
%token MOD // %

%token ADD_ASSIGN // +=
%token SUB_ASSIGN // -=
%token MUL_ASSIGN // *=
%token DIV_ASSIGN // /=
%token MOD_ASSIGN // %=
%token ASSIGN // =

%token AND // &&
%token OR  // ||

%token EQ     // ==
%token NOT_EQ // !=
%token NOT    // !

%token LT     // <
%token GT     // >
%token LTE    // <=
%token GTE    // >=

// Keywords
%token BREAK      // break
%token CONTINUE   // continue
%token DATA       // data
%token CONTRACT   // contract
%token IF       // if
%token ELIF     // elif
%token ELSE     // else
%token RETURN   // return
%token WHILE   // while
%token FUNC    // func
%token FOR     // for
%token IN      // in
%token SWITCH  // switch
%token CASE    // case
%token READ    // read
%token DEFAULT // default

// Types
%token T_INT    // int
%token T_BOOL   // bool
%token T_STR  // str
%token T_ARR  // arr
%token T_MAP  // map
%token T_FLOAT  // float
%token T_MONEY  // money
%token T_OBJECT  // object
%token T_BYTES  // bytes
%token T_FILE  // file

%type <i> ordinaltype
%type <n> type
%type <n> rettype
%type <sa> ident_list
%type <va> var_declaration
%type <va> var_declarations
%type <va> par_declaration
%type <va> par_declarations
%type <va> contract_data
%type <n> params
%type <n> var
%type <n> expr
%type <n> elif
%type <n> else
%type <n> switch
%type <n> case
%type <n> default
%type <n> statement
%type <n> statements
%type <n> cntparams
%type <n> contract_body
%type <n> contract_declaration
%type <n> index
%type <n> exprlist
%type <n> exprmaplist
%type <n> exprobj
%type <n> object
%type <n> objlist
%type <b> contract_read

%left AND 
%left OR
%left LTE GTE LT GT EQ NOT_EQ
%left ADD SUB
%left MUL DIV MOD
%right UNARYMINUS UNARYNOT

%start contract_declaration

%%

ordinaltype
    : T_BOOL {$$ = VBool}
    | T_INT {$$ = VInt}
    | T_STR {$$ = VStr}
    | T_ARR {$$ = VArr}    
    | T_MAP {$$ = VMap}    
    | T_FLOAT {$$ = VFloat}
    | T_MONEY {$$ = VMoney}
    | T_OBJECT {$$ = VObject}
    | T_BYTES {$$ = VBytes}    
    | T_FILE {$$ = VFile}    
    ;

type
    : ordinaltype {$$ = newType($1, yylex)}
    | type DOT ordinaltype {$$ = addSubtype($1, $3, yylex)}
    ;

rettype
    : /*empty*/ { $$ = nil }
    | type { $$ = $1 }
    ;

statements
    : /*empty*/ { $$ = nil }
    | statements NEWLINE { $$ = $1 }
    | statements switch { $$ = addStatement($1, $2, yylex)}
    | statements statement NEWLINE { $$ = addStatement($1, $2, yylex)}
    ;

params
    : /*empty*/ { $$ = nil }
    | expr { $$ = newParam( $1, yylex ) }
    | params COMMA expr { $$ = addParam($1, $3)}
    ;

cntparams
    : /*empty*/ { $$ = nil }
    | IDENT COLON expr { $$ = newContractParam( $1, $3, yylex ) }
    | cntparams COMMA IDENT COLON expr { $$ = addContractParam($1, $3, $5)}
    ;

var 
    : IDENT { $$ = newVarValue($1, yylex); }

index 
    : INDEX expr RBRACKET { $$ = newIndex($1, $2, yylex);}
    | index LBRACKET expr RBRACKET { $$ = addIndex($1, $3, yylex);}

else 
   : /*empty*/ {$$ = nil}
   | ELSE LBRACE statements RBRACE { $$ = $3 }
   ;

elif
   : /*empty*/ {$$ = nil}
   | elif ELIF expr LBRACE statements RBRACE { $$ = newElif($1, $3, $5, yylex) }
   ;

case
   : /*empty*/ {$$ = nil}
   | case CASE exprlist LBRACE statements RBRACE NEWLINE { $$ = newCase($1, $3, $5, yylex) }
   ;

default 
   : /*empty*/ {$$ = nil}
   | DEFAULT LBRACE statements RBRACE { $$ = $3 }
   ;

switch
    : SWITCH expr NEWLINE case default { $$ = newSwitch( $2, $4, $5, yylex ) }
    ;

statement 
    : var ASSIGN expr { $$ = newBinary($1, $3, ASSIGN, yylex) }
    | var ADD_ASSIGN expr { $$ = newBinary($1, $3, ADD_ASSIGN, yylex); }
    | var SUB_ASSIGN expr { $$ = newBinary($1, $3, SUB_ASSIGN, yylex); }
    | var MUL_ASSIGN expr { $$ = newBinary($1, $3, MUL_ASSIGN, yylex); }
    | var DIV_ASSIGN expr { $$ = newBinary($1, $3, DIV_ASSIGN, yylex); }
    | var MOD_ASSIGN expr { $$ = newBinary($1, $3, MOD_ASSIGN, yylex); } 
    | index ASSIGN expr { $$ = newBinary($1, $3, ASSIGN, yylex) }
    | type IDENT ASSIGN expr { $$ = newBinary( newVarDecl( $1, []string{$2}, yylex ), $4, ASSIGN, yylex) }
    | type ident_list { $$ = newVarDecl( $1, $2, yylex )}
    | IF expr LBRACE statements RBRACE elif else { $$ = newIf( $2, $4, $6, $7, yylex )}
    | BREAK { $$ = newBreak(yylex); }
    | CONTINUE { $$ = newContinue(yylex); }
    | RETURN { $$ = newReturn(nil, yylex); }
    | RETURN expr { $$ = newReturn($2, yylex); }
    | WHILE expr LBRACE statements RBRACE { $$ = newWhile( $2, $4, yylex )}
    | FUNC CALL par_declarations RPAREN rettype LBRACE statements RBRACE { 
           $$ = newFunc($2, $3, $5, $7, yylex)
           }
    | CALL params RPAREN { $$ = newCallFunc($1, $2, yylex);}
    | CALLCONTRACT cntparams RPAREN { $$ = newCallContract($1, $2, yylex);}
    | FOR IDENT IN expr LBRACE statements RBRACE { $$ = newFor( $2, $4, $6, yylex )}
    | FOR IDENT COMMA IDENT IN expr LBRACE statements RBRACE { $$ = newForAll( $2, $4, $6, $8, yylex )}
    | FOR IDENT IN expr DOUBLEDOT expr LBRACE statements RBRACE { $$ = newForInt( $2, $4, $6, $8, yylex )}
    ;

exprlist
    : expr { $$ = newArray($1, yylex); }
    | exprlist COMMA expr { $$ = appendArray($1, $3, yylex);}
    ;   

exprmaplist
    : STRING COLON expr { $$ = newMap($1, $3, yylex); }
    | exprmaplist COMMA STRING COLON NEWLINE expr { $$ = appendMap($1, $3, $6, yylex); }
    | exprmaplist COMMA STRING COLON expr { $$ = appendMap($1, $3, $5, yylex); }
    ;   

object
    : STRING COLON exprobj { $$ = newObj($1, $3, yylex); }
    | IDENT COLON exprobj { $$ = newObj($1, $3, yylex); }
    | object COMMA STRING COLON exprobj { $$ = appendObj($1, $3, $5, yylex);}
    | object COMMA IDENT COLON exprobj { $$ = appendObj($1, $3, $5, yylex);}
    ;   

objlist
    : exprobj { $$ = newObjArr($1, yylex); }
    | objlist COMMA exprobj { $$ = appendObjArr($1, $3, yylex);}
    ;   

exprobj
    : LPAREN expr RPAREN { $$ = $2; }
    | INT { $$ = newValue($1, yylex);}
    | FLOAT { $$ = newValue($1, yylex);}
    | STRING { $$ = newValue($1, yylex);}
    | QSTRING { $$ = newValue($1, yylex);}
    | TRUE { $$ = newValue(true, yylex);}
    | FALSE { $$ = newValue(false, yylex);}
    | CALL params RPAREN { $$ = newCallFunc($1, $2, yylex);}
    | CALLCONTRACT cntparams RPAREN { $$ = newCallContract($1, $2, yylex);}
    | index { $$ = $1}
    | ENV { $$ = newEnv($1, yylex);}
    | IDENT { $$ = newGetVar($1, yylex);}
    | LBRACE object RBRACE { $$ = $2;}
    | LBRACKET objlist RBRACKET { $$ = $2;}
    | LBRACKET object RBRACKET { $$ = newObjList($2, yylex);}


expr
    : LPAREN expr RPAREN { $$ = $2; }
    | INT { $$ = newValue($1, yylex);}
    | FLOAT { $$ = newValue($1, yylex);}
    | STRING { $$ = newValue($1, yylex);}
    | QSTRING { $$ = newValue($1, yylex);}
    | TRUE { $$ = newValue(true, yylex);}
    | FALSE { $$ = newValue(false, yylex);}
    | CALL params RPAREN { $$ = newCallFunc($1, $2, yylex);}
    | CALLCONTRACT cntparams RPAREN { $$ = newCallContract($1, $2, yylex);}
    | index { $$ = $1}
    | ENV { $$ = newEnv($1, yylex);}
    | IDENT { $$ = newGetVar($1, yylex);}
    | OBJ object RBRACE { $$ = $2;}
    | LBRACE exprlist RBRACE { $$ = $2;}
    | LBRACE exprmaplist RBRACE { $$ = $2;}
    | QUESTION LPAREN expr COMMA expr COMMA expr RPAREN { $$ = newQuestion($3, $5, $7, yylex);}
    | expr MUL expr { $$ = newBinary($1, $3, MUL, yylex); }
    | expr DIV expr { $$ = newBinary($1, $3, DIV, yylex);  }
    | expr ADD expr { $$ = newBinary($1, $3, ADD, yylex); }
    | expr SUB expr { $$ = newBinary($1, $3, SUB, yylex);}
    | expr MOD expr { $$ = newBinary($1, $3, MOD, yylex); } 
    | expr AND expr { $$ = newBinary($1, $3, AND, yylex); }
    | expr OR expr { $$ = newBinary($1, $3, OR, yylex);  }
    | expr EQ expr { $$ = newBinary($1, $3, EQ, yylex); }
    | expr NOT_EQ expr { $$ = newBinary($1, $3, NOT_EQ, yylex);}
    | expr LTE expr { $$ = newBinary($1, $3, LTE, yylex); }
    | expr GTE expr { $$ = newBinary($1, $3, GTE, yylex);  }
    | expr LT expr { $$ = newBinary($1, $3, LT, yylex); }
    | expr GT expr { $$ = newBinary($1, $3, GT, yylex);}

    | SUB expr %prec UNARYMINUS { $$ = newUnary($2, SUB, yylex)}
    | NOT expr %prec UNARYNOT { $$ = newUnary($2, NOT, yylex)}
    ;

ident_list
    : IDENT { $$ = []string{$1} }
    | ident_list IDENT { $$ = append($1, $2) }
    ;

par_declaration
    : type ident_list { $$ = newVars($1, $2)}
    ;

par_declarations
    : /*empty*/ {$$=nil}
    | par_declaration { $$ = $1}
    | par_declarations COMMA par_declaration { $$ = append($1, $3...) }
    ;

var_declaration
    : type ident_list { $$ = newVars($1, $2) }
    | type IDENT ASSIGN expr { $$ = newVarExp($1, $2, $4, yylex) }
    ;

var_declarations
    : var_declaration { $$=$1 }
    | var_declarations NEWLINE var_declaration { $$ = append($1, $3...) }
    ;

contract_data 
    : /*empty*/ { $$ = nil }
    | DATA LBRACE NEWLINE var_declarations NEWLINE RBRACE NEWLINE { $$ = $4 }
    ;

contract_body 
    : contract_data statements {
        $$ = newBlock($1, $2, yylex)
    }
    ;

contract_read
    : /*empty*/ { $$ = false }
    | READ { $$ = true }
    ;

contract_declaration 
    : CONTRACT IDENT contract_read LBRACE NEWLINE contract_body RBRACE { 
        $$ = newContract($2, $3, $6, yylex)
        setResult(yylex, $$)
        }
    | contract_declaration NEWLINE
    ;  
