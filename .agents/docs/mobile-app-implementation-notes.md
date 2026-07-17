# Mobile app implementation notes

This file records implementation decisions that close roadmap evaluation items without changing the
roadmap's product direction.

## Milestone 3

### Widget interaction

The first production widgets remain deep-link-only. A completion button would have to authenticate,
perform a network mutation against an arbitrary self-hosted server, report conflicts and expired
credentials, and reconcile WidgetKit and Glance failure UI while the main app is not active. Neither
platform gives that path feedback reliable enough for a household task to appear completed before
the server has accepted it. Widget rows therefore open the exact task; accepted in-app task writes
immediately rebuild the durable widget snapshot. Interactive actions can be reconsidered with an
idempotent offline-write protocol and explicit pending/failed states.

## Milestone 4

### Recipe editing

The native recipe editor uses explicit save and a compact ordered-text format for ingredient and
instruction sections. Section headings use `##`, ingredient lines use `quantity | item`, and
instruction lines use Markdown list markers. This keeps reordering direct and keyboard-friendly on
phones and tablets while the shared draft parser produces the same ordered API structures used by
web. Recipe descriptions and instruction bodies remain raw Markdown on the wire. Android and iOS
create, edit, and delete flows were exercised end to end; the adaptive editor and space activity
surface were also exercised in the iPad Simulator.
