# Diagrams

A Markdown file shows each ` ```mermaid ` code block as a diagram. Jun draws it itself, with [sumi](https://github.com/bencode/sumi), a Go implementation of Mermaid's layout: no browser, no JavaScript. Open this page in Jun's [Files](files.md) to see every diagram below drawn.

| Diagram | Starts with |
|---|---|
| [Sequence](#sequence) | `sequenceDiagram` |
| [Flowchart](#flowchart) | `flowchart`, `graph` |
| [State](#state) | `stateDiagram`, `stateDiagram-v2` |
| [Class](#class) | `classDiagram`, `classDiagram-v2` |
| [Entity relationship](#entity-relationship) | `erDiagram` |
| [Gantt](#gantt) | `gantt` |
| [Pie](#pie) | `pie` |

- **Colors** follow Jun's theme, light or dark.
- **Size**: a diagram is drawn as Mermaid lays it out, made smaller to fit the [reading width](files.md), never larger. A Gantt chart is laid out as wide as the reading width.
- **Not drawn**: a block sumi cannot draw stays code, with the reason below it. So does a block still being written, until its closing fence.
- **Not yet**: other diagram types, such as Git graphs, mind maps, timelines and user journeys, and LaTeX math. sumi's [feature checklist](https://github.com/bencode/sumi/blob/main/docs/compatibility.md) lists what each type draws.

## Sequence

Participants and actors, activations, notes, numbering, and `loop`, `alt`, `opt`, `par` and `critical` blocks.

```mermaid
sequenceDiagram
    autonumber
    actor U as You
    participant D as Deputy
    participant J as Jun
    participant C as Codex
    U->>D: Watch my agents
    D->>+J: watch
    loop each status change
        J-->>D: Codex: working → waiting
        D->>J: screen
        J-->>D: its last 30 lines
    end
    alt it asks you
        D->>J: message --kind ask
        Note over U,J: The ask shows in Messages in Global
        U->>C: Yes, go on
    else it finished
        D->>J: message --kind done
    end
    deactivate J
```

## Flowchart

Four directions, the classic shapes, labeled and dotted links, and subgraphs.

```mermaid
flowchart TB
    md[/"Markdown file"/] --> fence{"mermaid<br>block?"}
    fence -- no --> text["Shown as text"]
    fence -- yes --> sumi[["sumi lays it out"]]
    sumi --> drawn{"Supported?"}
    subgraph Jun
        drawn -- yes --> scene[("Scene")] --> paint(["Painted in the theme"])
        drawn -. no .-> code["Code, and the reason"]
    end
```

## State

Start and end, transitions with labels, composite states, and notes.

```mermaid
stateDiagram-v2
    [*] --> Idle
    Idle --> Working: a prompt
    Working --> Waiting: asks permission
    Waiting --> Working: you answer
    Working --> Idle: the reply ends
    Working --> [*]: terminal closed
    state Working {
        [*] --> Thinking
        Thinking --> Tool: a tool call
        Tool --> Thinking: its result
    }
    note right of Waiting: The deputy reports it as an ask
```

## Class

Members and methods, annotations, inheritance, realization, and associations with multiplicities.

```mermaid
classDiagram
    class Agent {
        <<interface>>
        +Name() string
        +Status() string
    }
    class Terminal {
        +int id
        +string project
        +string dir
        +Screen(lines int) string
    }
    class Message {
        +int id
        +string kind
        +string title
        +Update(text string)
    }
    Agent <|.. Claude
    Agent <|.. Codex
    Agent <|.. Pi
    Terminal "1" o-- "0..1" Agent : runs
    Message "*" --> "0..1" Terminal : links to
```

## Entity relationship

Entities with attributes, keys and comments, and every cardinality.

```mermaid
erDiagram
    PROJECT ||--o{ TERMINAL : has
    PROJECT ||--o{ PERSPECTIVE : arranges
    TERMINAL ||--o| AGENT : runs
    MESSAGE }o--o| TERMINAL : "links to"
    MESSAGE ||--o{ REPLY : threads
    PROJECT {
        string name PK
        string dir
    }
    TERMINAL {
        int id PK
        string project FK
        string title
    }
    MESSAGE {
        int id PK
        string kind "ask, warn, done or info"
        datetime time
    }
```

## Gantt

Sections, dependencies, done and active tasks, milestones, and skipped weekends.

```mermaid
gantt
    title A release
    dateFormat YYYY-MM-DD
    axisFormat %b %d
    excludes weekends
    section Build
        Design        :done, design, 2026-11-02, 3d
        Implement     :done, build, after design, 6d
        Review        :active, review, after build, 2d
    section Ship
        Fix findings  :fix, after review, 2d
        Install       :install, after fix, 1d
        Released      :milestone, after install, 0d
```

## Pie

A title, slices, and their values with `showData`.

```mermaid
pie showData
    title Where an agent's day goes
    "Working" : 52
    "Waiting for you" : 18
    "Idle" : 30
```
