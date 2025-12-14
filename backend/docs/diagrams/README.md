# Diagrammes d'architecture

## Prérequis

Installer [Graphviz](https://graphviz.org/)

## Générer les diagrammes

```bash
dot -Tsvg -Gbgcolor=transparent cloud-architecture.gv -o cloud-architecture.svg
dot -Tpng -Gbgcolor=transparent -Gdpi=150 cloud-architecture.gv -o cloud-architecture.png
```
