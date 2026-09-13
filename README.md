# ContextRail Prototype Workspace

This workspace contains the ContextRail product and planning material. The reusable Project Context Contract template is [template/](template/README.md); its optional [Spec Kit integration](template/integrations/spec-kit/README.md) maps specification-driven development into the governed artifacts. The first governed example Project is [Order Operations Portal](demo/order-operations-portal/README.md).

The demo is deliberately a separate Project fixture: it contains its own Context Contract documents, Go application, Request, DecisionRecord, engineering evidence and a pending Cloud Run staging receipt fixture. Local development does not require Cloud Run authorization; staging remains `NEEDS_INPUT` until real deployment access and a receipt exist. Production is represented as a protected decision threshold and is not deployed.
