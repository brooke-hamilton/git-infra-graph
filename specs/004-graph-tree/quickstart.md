# Quickstart: Recursive Tree Listing (`grif tree`)

<!-- markdownlint-disable MD013 -->

## Prerequisites

- A Git repository with at least one initialized graph (`grif init`)
- Some nodes created via `grif put`

## Setup (create sample data)

```bash
# Initialize a graph
grif init default

# Create some nodes
grif put default/network/vpc --data '{"cidr": "10.0.0.0/16"}'
grif put default/network/subnet --data '{"cidr": "10.0.1.0/24"}'
grif put default/compute/instance --data '{"type": "t3.micro"}'

# Commit the graph
grif commit default --message "Initial infrastructure"
```

## View full tree of a graph

```bash
$ grif tree default
default
├── compute
│   └── instance  (blob, c3d4e5f6)
└── network
    ├── subnet  (blob, e5f6a7b8)
    └── vpc  (blob, a1b2c3d4)
```

## View trees for all graphs

```bash
# Create a second graph
grif init staging
grif put staging/compute/instance --data '{"type": "t3.small"}'
grif commit staging

$ grif tree
default
├── compute
│   └── instance  (blob, c3d4e5f6)
└── network
    ├── subnet  (blob, e5f6a7b8)
    └── vpc  (blob, a1b2c3d4)

staging
└── compute
    └── instance  (blob, f7a8b9c0)
```

## View a subtree

```bash
$ grif tree default/network
network
├── subnet  (blob, e5f6a7b8)
└── vpc  (blob, a1b2c3d4)
```

## Limit depth

```bash
$ grif tree default --depth 1
default
├── compute
└── network

$ grif tree default --depth 0
default
```

## JSON output

```bash
$ grif tree --json default
{
  "name": "default",
  "type": "tree",
  "id": "d4e5f6a7",
  "children": [
    {
      "name": "compute",
      "type": "tree",
      "id": "b8c9d0e1",
      "children": [
        {
          "name": "instance",
          "type": "blob",
          "id": "c3d4e5f6"
        }
      ]
    },
    {
      "name": "network",
      "type": "tree",
      "id": "a7b8c9d0",
      "children": [
        {
          "name": "subnet",
          "type": "blob",
          "id": "e5f6a7b8"
        },
        {
          "name": "vpc",
          "type": "blob",
          "id": "a1b2c3d4"
        }
      ]
    }
  ]
}
```

## All graphs in JSON

```bash
$ grif tree --json
[
  {
    "name": "default",
    "type": "tree",
    "id": "d4e5f6a7",
    "children": [...]
  },
  {
    "name": "staging",
    "type": "tree",
    "id": "f6a7b8c9",
    "children": [...]
  }
]
```

## Error cases

```bash
# Graph not found
$ grif tree nonexistent
Error: graph 'nonexistent' not found

# No graphs
$ grif tree
Error: no graphs found

# Invalid depth
$ grif tree default --depth -1
Error: depth must be a non-negative integer
```
