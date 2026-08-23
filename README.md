# goas
The project is based on  
- [yvasiyarov/swagger](https://github.com/yvasiyarov/swagger) repository.  
- [uudashr/go-module](https://github.com/uudashr/go-module) repository. (currently deprecated)
- [mikunalpha/goas](https://github.com/mikunalpha/goas) repository.
- [nicocesar/goas](https://github.com/nicocesar/goas) repository.
- [launchdarkly/goas](https://github.com/launchdarkly/goas) repository. (currently archived)

Generate [OpenAPI Specification](https://swagger.io/specification) json file with comments in Go.

The public contract is intentionally small: the root package exposes the generation facade and a single legacy compatibility helper, while parsing, schema resolution, and package discovery live under internal packages. This keeps the OpenAPI model independent from orchestration and avoids parallel registries.

## Public API and compatibility

The supported public surface is the `Generator` API (`New()`, `Generate()`, and `GenerateTo()`) plus the `Options` configuration type. `NewParser` remains exported only for source compatibility with older integrations and is now deprecated: it is a legacy compatibility entry point and should not be used in new code. The underlying parser value is intentionally opaque because it is an internal implementation detail rather than a supported public type.

## Support and limitations

- Go modules are required. The module root must contain `go.mod`.
- Service-level annotations are read only from the selected entrypoint. When
  `--main-file-path` is empty, goas discovers a Go file in the module root
  whose package is `main`.
- Anonymous struct fields are supported when they embed another struct. Their
  properties are flattened into the containing schema; an existing property
  with the same name takes precedence.
- Built-in scalar types, structs, pointers, slices, maps, and interfaces are
  supported. Map keys are represented by OpenAPI object properties, so map
  values are the type that is documented.
- `time.Time`, `uuid.UUID`, and selected BSON types have special handling.
  Other external types depend on source being available through the Go
  toolchain and may only produce a partial schema.
- For responses, `object` keeps the resolved schema and `array` wraps it in an
  array when the schema is not already an array. Omitting the response type
  creates a response without a content schema, which is useful for `204`.
- `@ApiSchemaName` changes the component name. With `--omit-packages`, a
  conflict between types with the same name is an error.

## Install

```
go install github.com/delley/goas/cmd/goas@latest
```

## Usage

You can document your service by placing annotations inside your godoc at various places in your code.

### Service Description

The service description comments must be in the entrypoint Go file selected by `--main-file-path`. If that flag is omitted, goas looks for a `main` package file directly under the module root. They provide general information about the service you are documenting.

```go
// @Version 1.0.0
// @Title Backend API
// @Description API usually works as expected. But sometimes its not true.
// @ContactName Abcd
// @ContactEmail abce@email.com
// @ContactURL http://someurl.oxox
// @TermsOfServiceUrl http://someurl.oxox
// @LicenseName MIT
// @LicenseURL https://en.wikipedia.org/wiki/MIT_License
// @Server http://www.fake.com Server-1
// @Server http://www.fake2.com Server-2
// @Security AuthorizationHeader read write
// @SecurityScheme AuthorizationHeader http bearer Input your token
```

#### Security

If authorization is required, you must define security schemes and then apply those to the API.
A scheme is defined using `@SecurityScheme [name] [type] [parameters]` and applied by adding `@Security [scheme-name] [scope1] [scope2] [...]`. 

All examples in this section use `MyApiAuth` as the name. This name can be anything you chose; multiple named schemes are supported.
**Each scheme must have its own unique name.** If you attempt to define a security scheme with a name that already exists, the parser will reject it with an error message indicating the line number where the first definition was found.

A number of different types is supported, they all have different parameters:

|Type|Description|Parameters|Example|
|---|---|---|---|
|HTTP|A HTTP Authentication scheme using the `Authorization` header|scheme: any [HTTP Authentication scheme](https://www.iana.org/assignments/http-authschemes/http-authschemes.xhtml)|`@SecurityScheme MyApiAuth http basic`|
|APIKey|Authorization by passing an API Key along with the request|in: Location of the API Key, options are `header`, `query` and `cookie`. name: The name of the field where the API Key must be set|`@SecurityScheme MyApiAuth apiKey header X-MyCustomHeader`|
|OpenIdConnect|Delegating security to a known OpenId server|url: The URL of the OpenId server|`@SecurityScheme MyApiAuth openIdConnect https://example.com/.well-known/openid-configuration`|
|OAuth2AuthCode|Using the "Authentication Code" flow of OAuth2|authorizationUrl, tokenUrl|`@SecurityScheme MyApiAuth oauth2AuthCode /oauth/authorize /oauth/token`| 
|OAuth2Implicit|Using the "Implicit" flow of OAuth2|authorizationUrl|`@SecurityScheme MyApiAuth oauth2Implicit /oauth/authorize`|
|OAuth2ResourceOwnerCredentials|Using the "Resource Owner Credentials" flow of OAuth2|tokenUrl|`@SecurityScheme MyApiAuth oauth2ResourceOwnerCredentials /oauth/token`|
|OAuth2ClientCredentials|Using the "Client Credentials" flow of OAuth2|tokenUrl|`@SecurityScheme MyApiAuth oauth2ClientCredentials /oauth/token`|

Any text that is present after the last parameter will be used as the description. For instance `@SecurityScheme MyApiAuth http basic Login with your admin credentials`.

##### Security Scheme Rules

**Unique Names**: Each security scheme name must be unique across your API. If you attempt to redefine a scheme with the same name, the parser will reject it with an error message indicating where the first definition was found. Example error:
```
security scheme 'MyOAuth2' already defined at line 5
```

##### OAuth2 Flow Examples

The OAuth2 standard defines four authorization flows, each suited for different scenarios:

**1. Authorization Code Flow** (most common for web applications):
```go
// @SecurityScheme OAuth2 oauth2AuthCode https://example.com/oauth/authorize https://example.com/oauth/token
// @Security OAuth2 read write
```
This flow requires both an authorization URL (where users log in) and a token URL (where the app exchanges authorization for an access token).

**2. Implicit Flow** (for browser-based applications):
```go
// @SecurityScheme OAuth2Implicit oauth2Implicit https://example.com/oauth/authorize
// @Security OAuth2Implicit read
```
This flow requires only an authorization URL. The access token is returned directly from the authorization endpoint.

**3. Resource Owner Password Credentials Flow** (for trusted applications):
```go
// @SecurityScheme OAuth2ResourceOwner oauth2ResourceOwnerCredentials https://example.com/oauth/token
// @Security OAuth2ResourceOwner read write
```
This flow requires only a token URL. The app directly requests tokens using user credentials (not recommended for untrusted clients).

**4. Client Credentials Flow** (for server-to-server communication):
```go
// @SecurityScheme OAuth2Client oauth2ClientCredentials https://example.com/oauth/token
// @Security OAuth2Client admin
```
This flow requires only a token URL. The app authenticates directly using its own credentials, not a user's credentials.

Once all security schemes have been defined, they must be configured. This is done with the `@Security` comment.
Depending on the `type` of the scheme, scopes (see below) may be supported. *At the moment, it is only possible to configure security for the entire service*.

```go
// @Security MyApiAuth read_user write_user
```

##### Scopes
For OAuth2 security schemes, it is possible to define scopes using the `@SecurityScope [schema-name] [scope-code] [scope-description]` comment. Scopes represent specific permissions that a user can grant to an application.

```go
// @SecurityScope OAuth2 read_user Read user profile information
// @SecurityScope OAuth2 write_user Create and modify user profile
// @SecurityScope OAuth2 read_posts Read posts and comments
// @SecurityScope OAuth2 write_posts Create and modify posts
```

Scopes are associated with the security scheme they belong to. When you apply a security scheme to your API using `@Security`, the scopes you specify must exist in the corresponding scheme's scope definitions. For example:

```go
// Define the scheme with its scopes
// @SecurityScheme OAuth2 oauth2AuthCode https://example.com/oauth/authorize https://example.com/oauth/token
// @SecurityScope OAuth2 read_user Read user profile information
// @SecurityScope OAuth2 write_user Create and modify user profile

// Apply the scheme to your API, requesting specific scopes
// @Security OAuth2 read_user write_user
```

This tells API documentation tools like Swagger/OpenAPI UI that users must grant both the `read_user` and `write_user` scopes when authenticating with your API.

### Handler funcs

By adding comments to your handler func godoc, you can document individual actions as well as their input and output.

```go
type User struct {
  ID   uint64 `json:"id" example:"100" description:"User identity"`
  Name string `json:"name" example:"Mikun"` 
}

type UsersResponse struct {
  Data []Users `json:"users" example:"[{\"id\":100, \"name\":\"Mikun\"}]"`
}

type Error struct {
  Code string `json:"code"`
  Msg  string `json:"msg"`
}

type ErrorResponse struct {
  ErrorInfo Error `json:"error"`
}

// @Title Get user list of a group.
// @Description Get users related to a specific group.
// @Param  groupID  path  int  true  "Id of a specific group."
// @Success  200  object  UsersResponse  "UsersResponse JSON"
// @Failure  400  object  ErrorResponse  "ErrorResponse JSON"
// @Resource users
// @Route /api/group/{groupID}/users [get]
func GetGroupUsers() {
  // ...
}

// @Title Get user list of a group.
// @Description Create a new user.
// @Param  user  body  User  true  "Info of a user."
// @Success  200  object  User           "UsersResponse JSON"
// @Failure  400  object  ErrorResponse  "ErrorResponse JSON"
// @Resource users
// @Route /api/user [post]
func PostUser() {
  // ...
}
```

#### Title & Description
```
@Title {title}
@Title Get user list of a group.

@Description {description}.
@Description Get users related to a specific group.
```
- {title}: The title of the route.
- {description}: The description of the route.

#### Parameter
```
@Param  {name}  {in}  {goType}  {required}  {description}
@Param  user    body  User      true        "Info of a user."
```
- {name}: The parameter name.
- {in}: The parameter is in `path`, `query`, `form`, `header`, `cookie`, `body` or `file`.
- {goType}: The type in go code. This will be ignored when {in} is `file`.
- {required}: `true`, `false`, `required` or `optional`. 
- {description}: The description of the parameter. Must be quoted.

##### allOf, anyOf, oneOf, not

`allOf`, `anyOf`, `oneOf` and `not` are supported for `body` parameter types. `not` requires a single argument, while the others require one or more. e.g.

```
@Param user body oneOf(User,string) true "Info of a user."
```

#### Response
```
@Success  {status}  {jsonType}  {goType}       {description}
@Success  200       object      UsersResponse  "UsersResponse JSON"

@Failure  {status}  {jsonType}  {goType}       {description}
@Failure  400       object      ErrorResponse  "ErrorResponse JSON"
```
- {status}: The HTTP status code.
- {jsonType}: Optional. Use `object` or `array`; `array` produces an array
  schema around {goType} when needed. Omit it for a response without a
  content schema, such as `204`.
- {goType}: The type in go code.
- {description}: The description of the response. Must be quoted.

#### Resource & Tag
```
@Resource {resource}
@Resource users

@Tags "{tag}" ["{description}"]
@Tags "users" "Operations related to users"
```
- {resource}, {tag}: Tag of the route.
- `@Tags` requires quoted names; a description is optional and must also be quoted.

#### Route
```
@Route {path}    {method}
@Route /api/user [post]

@Router {path}   {method}
@Router /api/user [post]
```
- {path}: The URL path.
- {method}: The HTTP Method. Must be put in brackets.
- `@Router` is kept as a compatibility alias of `@Route` and is accepted in the parser.

### Documentation Generation

Paths passed to `--main-file-path`, `--handler-path` and `--file-ref-path` are resolved relative to `--module-path` (which defaults to `.`). The output path is resolved relative to the directory where the command is run. The default output is `oas.json`; use `--output -` to write the generated JSON to stdout.

From the repository root, generate the included example to stdout:
```
go run ./cmd/goas --module-path ./example --main-file-path ./main.go --omit-packages --output -
```

From a module whose `go.mod` and main file are in the same directory:
```
goas --module-path . --output oas.json
```

From a module whose main file is in a different directory:
```
goas --module-path . --main-file-path ./cmd/xxx/main.go --output oas.json
```

When running the example as a separate Go module, build the binary once at the repository root and invoke it from the example directory:
```
go build -o ./goas-cli ./cmd/goas
cd example
../goas-cli --module-path . --main-file-path ./main.go --omit-packages --output ./example.json
```

The `go run ../cmd/goas` form is not valid from inside the `example` module because the command package is outside the active module. The entrypoint file must be under the module root or be provided explicitly with `--main-file-path`; service-level comments in other Go files are not read as global comments.

### Command-line options

|Option|Default|Description|
|---|---|---|
|`--module-path`|`.`|Module root to scan for annotations and `go.mod`|
|`--main-file-path`|empty|Entrypoint Go file; when empty, auto-detect a `main` package file in the module root|
|`--handler-path`|empty|Restrict handler comments to this path|
|`--file-ref-path`|`.`|Base path for file references|
|`--output`|`oas.json`|Output file, or `-` for stdout|
|`--debug`|`false`|Show debug messages|
|`--omit-packages`|`false`|Omit packages from schema names|
|`--show-hidden`|`false`|Include paths in hidden packages|

Description references using `$ref:` may point to local files or HTTP(S)
URLs. HTTP(S) references use a 30-second timeout when the generation context
has no deadline and are limited to 1 MiB responses; non-2xx responses and
other URL schemes fail generation. Local `file://` references are resolved
under `--file-ref-path`; paths and symlinks that escape that directory are
rejected.

## Local Development

### Unit tests

The example is a separate Go module. Run `go mod tidy` in the `example`
directory when its dependencies need to be synchronized.

Run `go test` in root directory to run unit tests.

### Build and run

Build the command package explicitly. The output name must not be `goas`, because `goas` is also the library directory in the repository:

```
go build -o ./goas-cli ./cmd/goas
```

From the repository root, run it against the included `example` application:

```
./goas-cli --module-path ./example --main-file-path ./main.go --omit-packages --output /tmp/goas-example.json
```

Alternatively, from the `example` directory run the binary from its parent:

```
../goas-cli --module-path . --main-file-path ./main.go --omit-packages --output ./example.json
```

The module path is the base for relative `--main-file-path`, `--handler-path` and
`--file-ref-path` values. Module dependencies are resolved using the Go toolchain's
`GOMODCACHE` (`go env GOMODCACHE`); the cache directory does not need to exist when
the generator starts. A cancelled generation context returns `context.Canceled` or
`context.DeadlineExceeded` and does not write output.