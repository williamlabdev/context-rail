# Cloud Run 部署手冊 — 2026-10-06 執行版

狀態：草案，供 10/6 GCP 專案／billing／`GEMINI_API_KEY` 到位當天照做
更新：2026-09-27（補上第 2 節步驟 3.1：Cloud Build 執行身份對 runtime SA 的 `roles/iam.serviceAccountUser` 綁定，這是 verifier 抓到的缺口，之前只做到「建立 runtime SA」沒做到「授權 Cloud Build 去用它」）；2026-09-27（`cloudbuild.yaml` / `scripts/deploy-cloud-run.sh` 已補上 `--set-secrets` 與 `--service-account`，見第 2 節步驟 4、第 3 節）；2026-09-27（`internal/change/advisor.go:136` 預設的 Gemini model `gemini-2.0-flash` 已被 Google 官方列為 deprecated，shutdown date 2026-06-01，官方建議的替代 model 是 `gemini-3.6-flash`——來源：[Gemini API model deprecations](https://ai.google.dev/gemini-api/docs/deprecations)、[Gemini API models](https://ai.google.dev/gemini-api/docs/models)；`cloudbuild.yaml` / `scripts/deploy-cloud-run.sh` 現在都沒有能傳遞 `CONTEXT_RAIL_GEMINI_MODEL` 的旗標，這次任務範圍不改這兩個檔案，改成第 2 節新增步驟 5.5 的部署後手動 `gcloud run services update`，見第 1 節該列、第 2 節步驟 5.5、第 2 節步驟 6、第 7 節風險 9——**手動指令未實測**）
前置演練：本機 `docker build` + `docker run` + curl，見〈本次本地演練結果〉一節；**未執行任何 `gcloud` 寫入動作、未 push image、未建立任何雲端資源**。0927 這次追加的變更只做了假 `gcloud` shim dry-run（見第 3 節），**同樣未跑過真的 `gcloud`／`docker`**。

> 這份手冊記錄的是「10/6 當天要做什麼」。既有的 [`CLOUD_RUN_BASELINE.md`](CLOUD_RUN_BASELINE.md) 是穩態容器契約文件（環境變數表、smoke check 定義），這份是一次性的執行清單，兩者對照著看；本手冊完成後，`CLOUD_RUN_BASELINE.md` 不需要改，但 `evidence/EB-008`、`evidence/EB-009`、`README.md:9` 的宣告值需要回填（見下方對應段落）。

## 0. 這次演練實際查證了什麼

讀了 `scripts/deploy-cloud-run.sh`、`Dockerfile`、`cloudbuild.yaml`、`scripts/bootstrap-w1.sh`、`scripts/record-promotion.sh`、`cmd/context-rail/main.go`、`internal/change/advisor.go`、`internal/change/service.go`、`.github/workflows/ci.yml`、`docs/operations/CLOUD_RUN_BASELINE.md`、`docs/operations/REAL_RUN_PLAYBOOK.md`。本機用 `docker build --platform=linux/amd64` 建置 image、`docker run` 起容器、curl 打 `/healthz`、`/`、`/v1/projects`、`POST /v1/projects`，全部符合預期（見第 5 節）。**沒有**安裝 `gcloud`（本機 `command not found: gcloud`），因此 Artifact Registry／Cloud Build／Cloud Run／Secret Manager 相關指令全部**未實測**，旗標是否完全正確以 10/6 當天實跑為準。

---

## 1. 前置清單 — 10/6 需要 William 提供／建立的 GCP 值

| 值 | 用途 | 出處 | 目前狀態 |
| --- | --- | --- | --- |
| GCP 專案 id（`PROJECT_ID`） | 所有資源的容器；`scripts/deploy-cloud-run.sh` 第一個必填參數 | `scripts/deploy-cloud-run.sh:30`（`PROJECT_ID="${1:?usage: ...}"`）、`scripts/bootstrap-w1.sh:18` | 未建立；`bootstrap-w1.sh:78` 若專案不存在會用 `gcloud projects create` 建立，名稱寫死 `ContextRail` |
| Billing account 已連結該專案 | `gcloud services enable` 與 Cloud Build 都需要 billing | `scripts/bootstrap-w1.sh:82-97`（讀 `gcloud billing accounts list --filter=open=true`，只有剛好一個帳號時自動連結，否則印出來要求手動 `gcloud billing projects link`） | 待 10/6 |
| Region（預設 `asia-east1`） | Artifact Registry、Cloud Run 部署區域 | `scripts/deploy-cloud-run.sh:31`、`cloudbuild.yaml:12`、`scripts/bootstrap-w1.sh:19` | 有預設值，可延用 |
| Cloud Run service 名稱（預設 `context-rail-staging`） | staging 服務名稱 | `scripts/deploy-cloud-run.sh:32` | 有預設值 |
| Artifact Registry repo 名稱（寫死 `context-rail`） | 存放 image | `scripts/deploy-cloud-run.sh:33`、`cloudbuild.yaml:13` | 腳本自動 `describe` 沒有就 `create`（`scripts/deploy-cloud-run.sh:49-51`） |
| 部署身份（操作機器上 `gcloud auth login` 的帳號） | 需要能 enable API、建 Artifact Registry repo、跑 Cloud Build、部署 Cloud Run | `scripts/deploy-cloud-run.sh:35`（只檢查 `command -v gcloud`，**沒有檢查權限**）、`scripts/bootstrap-w1.sh:70-74` | **未定義**——目前所有腳本都假設「當前 gcloud 使用者已有夠大的權限」，沒有指定或建立專用部署用 service account。10/6 若用 William 個人帳號（Owner/Editor）最省事，但要留意這代表本地開發機持有專案級高權限憑證 |
| Cloud Run **執行期**服務帳號（runtime SA） | 容器實際跑起來後用什麼身份呼叫 GCP API（目前程式碼完全不呼叫其他 GCP API，但要讀 Secret Manager 的 `GEMINI_API_KEY` 需要這個身份有 `roles/secretmanager.secretAccessor`） | `cloudbuild.yaml`（`deploy-staging` step）現在**要求** `_RUN_SERVICE_ACCOUNT` substitution / `scripts/deploy-cloud-run.sh` 要求 `RUN_SERVICE_ACCOUNT` 環境變數（0927 補上，見第 3 節） | **旗標已補上，但 SA 本身仍要手動建立**——沒設就清楚報錯、不會默默落到 Compute 預設 SA（`--service-account` 一定會帶值或直接失敗）。SA 的建立與 `secretAccessor` 綁定還是第 2 節步驟 3 的手動前置作業 |
| `GEMINI_API_KEY` 的值 | 啟用 `internal/change/advisor.go` 的 `GeminiAdvisor`；不設就 fallback 成 `RuleAdvisor` | `cmd/context-rail/main.go:34,150-152`、`internal/change/advisor.go:148-149`（`"GEMINI_API_KEY not configured"`）、`docs/operations/CLOUD_RUN_BASELINE.md:35` | William 提供；`cloudbuild.yaml` / `scripts/deploy-cloud-run.sh` 已補上 `--set-secrets`（`_GEMINI_SECRET` / `GEMINI_SECRET`，預設 secret 名稱 `gemini-api-key`，0927 補上，見第 3 節）——**secret 本身的建立與 IAM 綁定仍是第 2 節步驟 3 的手動前置作業** |
| `CONTEXT_RAIL_GEMINI_MODEL`（10/6 起建議必設） | 覆寫預設 model | `cmd/context-rail/main.go:35`、`internal/change/advisor.go:134-136`（預設 `gemini-2.0-flash`） | **預設值 `gemini-2.0-flash` 已被 Google 官方列為 deprecated**（shutdown date 2026-06-01，[Gemini API model deprecations](https://ai.google.dev/gemini-api/docs/deprecations)），官方目前建議的替代 model 是 `gemini-3.6-flash`（同頁面 + [Gemini API models](https://ai.google.dev/gemini-api/docs/models)）。`cloudbuild.yaml` / `scripts/deploy-cloud-run.sh` 目前都**沒有**能傳這個值的旗標（查過兩個檔案，`_GEMINI_SECRET`/`_RUN_SERVICE_ACCOUNT` 之外沒有其他 substitution 或 env-var 參數），這次任務範圍不改這兩個檔案，10/6 部署後改用第 2 節步驟 5.5 的手動 `gcloud run services update --update-env-vars`（**未實測**） |
| `GITHUB_TOKEN`（可選） | 啟用 candidate 的 GitHub read-back（compare / PR reviews / check runs） | `cmd/context-rail/main.go:36`、`docs/operations/CLOUD_RUN_BASELINE.md:37` | 本次任務範圍**不需要**——demo 用固定 fixture，不牽涉真實 repo read-back；除非 10/6 要順便展示 `REAL_RUN_PLAYBOOK.md` 的流程 |
| Firestore／Storage | 目前**完全沒有用到** | 全 repo 搜尋 `firestore`／`storage` 只有一處註解：`internal/topology/store.go:23`（`// persistence decision (files first, Firestore later)` ——純規劃註記，非程式碼依賴） | 不需要 10/6 準備任何 Firestore/Storage 資源 |
| 容器內部狀態目錄（不是 GCP 值，是風險提醒） | Governance ledger（Change / Candidate / Release / Topology 版本）存放位置 | `Dockerfile:34-41`（`CONTEXT_RAIL_STATE_DIR=/tmp/context-rail-state`，註解明講「不是持久化儲存」） | Cloud Run instance 重啟／scale-to-zero／重新部署都會清空；`cloudbuild.yaml:53` 目前 `--min-instances=0`，demo 過程中閒置太久有被 Google 回收、狀態消失的風險（見第 6 節風險） |

---

## 2. 逐步指令（10/6 當天）

以下每一步都附**預期輸出**與**失敗時的處置**。標「未實測」的旗標是照著 `cloudbuild.yaml` / `deploy-cloud-run.sh` 既有內容或 GCP 官方文件推導，10/6 當天以實際輸出為準，不要假裝已經驗證過。

### 步驟 0 — 安裝與登入（操作機器）

```sh
brew install --cask google-cloud-sdk   # 或既有安裝
gcloud auth login
gcloud auth application-default login  # Cloud Build / client library 認證用
```

**預期**：瀏覽器登入完成，`gcloud auth list --filter=status:ACTIVE --format='value(account)'` 印出登入帳號。
**失敗處置**：登入失敗多半是網路或帳號問題，重跑 `gcloud auth login`；不要在無人值守腳本裡塞帳密。

### 步驟 1 — 建立／選定專案與 billing

可直接用既有腳本（`scripts/bootstrap-w1.sh:69-97`）跑到 GCP 這段，或手動：

```sh
gcloud projects create "${PROJECT_ID}" --name="ContextRail" --set-as-default   # 專案不存在才需要
gcloud config set project "${PROJECT_ID}"
gcloud billing accounts list --filter=open=true
gcloud billing projects link "${PROJECT_ID}" --billing-account="<ACCOUNT_ID>"
```

**預期**：`gcloud billing projects describe "${PROJECT_ID}" --format='value(billingEnabled)'` 回傳 `True`。
**失敗處置**：billing account 不只一個時 `bootstrap-w1.sh:88-96` 會直接印出清單並退出，需要人工指定 `--billing-account`；沒有任何開通的 billing account 就必須先在 Cloud Console 開一個，這不是腳本能自動做的。

### 步驟 2 — 啟用 API、建 Artifact Registry repo

```sh
gcloud services enable run.googleapis.com cloudbuild.googleapis.com artifactregistry.googleapis.com secretmanager.googleapis.com
gcloud artifacts repositories describe context-rail --location="${REGION}" \
  || gcloud artifacts repositories create context-rail --location="${REGION}" \
       --repository-format=docker --description="ContextRail images"
```

對應 `scripts/deploy-cloud-run.sh:46-51`，唯一差異是**多加了 `secretmanager.googleapis.com`**——原腳本沒有這行，是本次演練發現的落差（原腳本從沒打算讀 Secret Manager）。
**預期**：`describe` 回傳 repo 詳細資訊，或 `create` 成功建立。
**失敗處置**：權限不足（`PERMISSION_DENIED`）代表步驟 0 的帳號沒有 `roles/artifactregistry.admin` 或等效權限；換帳號或加角色。

### 步驟 3 — 建立最小權限的 Cloud Run 執行身份（手動前置作業，`cloudbuild.yaml` 不會代為建立）

`cloudbuild.yaml` 的 `deploy-staging` step（0927 起）已經**要求**帶 `--service-account`，不會再默默落到專案預設的 Compute Engine SA——但這個旗標只是「指定用哪個身份」，SA 本身、以及它對 Secret Manager 的存取權，仍然要在 10/6 當天手動建立：

```sh
gcloud iam service-accounts create context-rail-run \
  --display-name="ContextRail Cloud Run runtime"
gcloud secrets create gemini-api-key --replication-policy=automatic   # 若尚未建立
echo -n "${GEMINI_API_KEY}" | gcloud secrets versions add gemini-api-key --data-file=-
gcloud secrets add-iam-policy-binding gemini-api-key \
  --member="serviceAccount:context-rail-run@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

**未實測**：以上 IAM/Secret Manager 指令是依 GCP 官方文件慣例寫的，10/6 當天第一次跑務必看清楚錯誤訊息，不要照抄後假設成功。
**失敗處置**：`gemini-api-key` 密鑰值請勿用 shell history 留痕（用 `echo -n ... | gcloud secrets versions add` 且該行結束後 `history -d` 或直接不留在互動 shell）；若 secret 已存在，`create` 會報 `ALREADY_EXISTS`，改用 `gcloud secrets versions add` 即可。

#### 步驟 3.1（0928 補上）— Cloud Build 的執行身份要能 `actAs` 這個 runtime SA，否則 `gcloud run deploy` 直接 `PERMISSION_DENIED`

上面建立的 `context-rail-run` 只是「Cloud Run 服務跑起來後」的身份。真正下 `--service-account=context-rail-run@...` 這個旗標的，是**執行 `deploy-staging` step 的那個 Cloud Build 身份**（不是你本機登入的帳號）——GCP 規定：要把服務／revision 綁到某個 SA 上，呼叫端本身要對那個目標 SA 有 `roles/iam.serviceAccountUser`（內含 `iam.serviceAccounts.actAs` 權限），且這個 binding 是綁在**目標 SA 資源本身**上，不是綁在專案層級（來源：[Cloud Run — Configure the service identity](https://cloud.google.com/run/docs/configuring/services/service-accounts)：「To get the permissions that you need to attach a service account as the service identity on the service or revision, you or your administrator must grant your deployer account the Service Account User role (`roles/iam.serviceAccountUser`) on the service account that is used as the service identity.」）。少這一步，`gcloud builds submit` 會在 `deploy-staging` step 卡住，訊息含 `PERMISSION_DENIED` 且提到 `iam.serviceAccounts.actAs`。

**先確認你的專案實際用哪個身份跑 Cloud Build**——2024 年之後新建的專案，Cloud Build 預設可能不是走 legacy 的 `<PROJECT_NUMBER>@cloudbuild.gserviceaccount.com`，而是走 Compute Engine 預設 SA `<PROJECT_NUMBER>-compute@developer.gserviceaccount.com`（依專案的組織政策設定而定；來源：[Cloud Build service accounts](https://cloud.google.com/build/docs/cloud-build-service-account)：「Depending on your organization's settings, Cloud Build may use the Compute Engine default service account or the legacy Cloud Build service account to execute builds on your behalf.」），兩者要綁的 member 不一樣，不要憑記憶假設是哪一個：

```sh
gcloud builds get-default-service-account [--region=REGION]
```

**已查證**：`gcloud builds get-default-service-account` 這個指令確實存在（[官方 gcloud 參考文件](https://cloud.google.com/sdk/gcloud/reference/builds/get-default-service-account)：「Get the default service account for a project.」），回傳的就是這個專案 Cloud Build 目前實際會用的服務帳號 email——把這一步的輸出，原封不動當成下面 `--member` 的值：

```sh
BUILD_SA="$(gcloud builds get-default-service-account --format='value(serviceAccountEmail)')"
gcloud iam service-accounts add-iam-policy-binding \
  "context-rail-run@${PROJECT_ID}.iam.gserviceaccount.com" \
  --member="serviceAccount:${BUILD_SA}" \
  --role="roles/iam.serviceAccountUser"
```

**未實測**：`get-default-service-account` 的輸出格式、以及上面這條 binding 在本專案是否真的補齊了 `PERMISSION_DENIED`，都要 10/6 當天用真的 `gcloud` 才能驗——這是本次任務新補的文件，指令本身沒有跑過。
**失敗處置**：如果 `get-default-service-account` 回傳的是 legacy `@cloudbuild.gserviceaccount.com`，也可能代表 Cloud Build API 是舊專案較早啟用；不論是哪一個，都用它回傳的 email 當 `--member`，不要自己猜專案編號組字串。

### 步驟 4 — secret 與 SA 現在已經寫進 `cloudbuild.yaml` / `scripts/deploy-cloud-run.sh`（0927 更新，不用再補手動指令）

之前這裡是「手動加 `--set-secrets` / `--service-account` 再跑」；0927 已經把這兩個旗標直接寫進 `cloudbuild.yaml` 的 `deploy-staging` step 與 `scripts/deploy-cloud-run.sh`，10/6 當天**不需要**再手動組這行 `gcloud run deploy`，改成在跑步驟 5 之前設好這些環境變數：

```sh
export RUN_SERVICE_ACCOUNT=context-rail-run   # 必填，短名稱不含 @project；腳本會自動組成 <name>@<project>.iam.gserviceaccount.com
export GEMINI_SECRET=gemini-api-key           # 選填，預設就是這個值（Secret Manager 的 secret 名稱，用 latest 版本）
# 如果 secret 還沒建（步驟 3 還沒跑完），要先用 fallback 部署：
# export SKIP_GEMINI_SECRET=1
```

`scripts/deploy-cloud-run.sh` 會把 `RUN_SERVICE_ACCOUNT` / `GEMINI_SECRET` 轉成 `gcloud builds submit --substitutions=...,_RUN_SERVICE_ACCOUNT=...,_GEMINI_SECRET=...`，`cloudbuild.yaml` 的 `deploy-staging` step 再組出：

```sh
gcloud run deploy context-rail-staging \
  --image="${IMAGE}@${DIGEST}" \
  --region="${REGION}" \
  --platform=managed \
  --port=8080 \
  --allow-unauthenticated \
  --min-instances=0 --max-instances=2 \
  --memory=256Mi --cpu=1 \
  --service-account="context-rail-run@${PROJECT_ID}.iam.gserviceaccount.com" \
  --set-secrets="GEMINI_API_KEY=gemini-api-key:latest" \
  --labels=context-rail-environment=staging,context-rail-commit="${SHORT_SHA}"
```

**行為細節（0927 補上，實測方式見第 3 節）**：
- `RUN_SERVICE_ACCOUNT` 沒設會直接報錯退出（腳本層與 `cloudbuild.yaml` 的 `deploy-staging` step 都各自擋一次），訊息會提示怎麼設，不會默默落到 Compute 預設 SA。
- `GEMINI_SECRET` 設空字串，或腳本設 `SKIP_GEMINI_SECRET=1`，就不帶 `--set-secrets`，部署 fallback 成 rule-advisor（在 secret 還沒建好的當下很有用）。
- 這兩個旗標的**參數組裝**已經用假 `gcloud` shim 做過 dry-run 驗證（見第 3 節）；`--set-secrets` 的語法（`ENV_VAR=SECRET_NAME:VERSION`）本身、以及 Cloud Run 是否真的把它掛載成環境變數，**仍然是 10/6 當天要用真的 `gcloud` 驗的事**——10/6 第一次跑完務必核對 `gcloud run services describe context-rail-staging --region="${REGION}" --format=yaml` 裡真的看得到這個環境變數是從 secret 掛載，不是空的。

### 步驟 5 — Build / Push / Deploy（照既有腳本，0927 起多兩個必要／選填環境變數）

```sh
RUN_SERVICE_ACCOUNT=context-rail-run \
  scripts/deploy-cloud-run.sh "${PROJECT_ID}" "${REGION}" context-rail-staging
```

`RUN_SERVICE_ACCOUNT` 是必填（見步驟 4）；不設 `GEMINI_SECRET` 就用預設值 `gemini-api-key`，secret 還沒建好就加 `SKIP_GEMINI_SECRET=1`。

這一行等同跑完 `scripts/deploy-cloud-run.sh` 全部內容：驗證 `RUN_SERVICE_ACCOUNT` 有設 → `gcloud config set project` → enable API（`run` / `cloudbuild` / `artifactregistry` / `secretmanager`）→ ensure AR repo → `gcloud builds submit --config cloudbuild.yaml`（帶入 `_RUN_SERVICE_ACCOUNT` / `_GEMINI_SECRET`，雲端建置，天然是 linux/amd64，不受本機 Apple Silicon 影響）→ 讀回 `url` / `revision` / `image` → 跑 `smoke()`（`/healthz` 要含 `"status":"ok"`、`/v1/projects` 要有 ≥2 個 fixture project、`/` 要 200、`POST /v1/projects` 要 405）。

`--service-account` 與（除非明確關閉）`--set-secrets` 現在會直接跟著這一行帶入，不需要再補步驟 4 的手動 `gcloud run services update`。

**預期輸出**（`scripts/deploy-cloud-run.sh:61-70`）：

```
== deployed
url:      https://context-rail-staging-<hash>-<region>.a.run.app
revision: context-rail-staging-00001-xxx
image:    <region>-docker.pkg.dev/<project>/context-rail/context-rail@sha256:<digest>
commit:   <short-sha>
```

**失敗處置**：
- Cloud Build 失敗最常見是 `npm ci` 或 `go build` 在雲端環境差異（例如 lockfile 不同步）——先看 Cloud Build log（`gcloud builds log <BUILD_ID>`）。
- `gcloud run deploy` 失敗且訊息含 `PERMISSION_DENIED` on secret：步驟 3 的 IAM binding 沒生效或打錯 SA email。
- smoke 失敗在 `/healthz`：容器沒有正確監聽 `$PORT`（理論上不會，`cmd/context-rail/main.go:207-214` 的 `listenAddress` 已經處理，本次本地演練也驗證過）；失敗在 `/v1/projects` 缺 fixture：image 沒把 `demo/order-operations-portal` / `examples/support-insights` 複製進去（檢查 `.dockerignore` 有沒有誤擋，見第 6 節）。

### 步驟 5.5 — 手動把 `CONTEXT_RAIL_GEMINI_MODEL` 設成現行模型（0927 新增，`cloudbuild.yaml` / `scripts/deploy-cloud-run.sh` 都不改，這次只補手動指令）

`internal/change/advisor.go:134-136` 的預設值 `gemini-2.0-flash` 已被 Google 官方列為 deprecated（shutdown date 2026-06-01，見第 1 節該列與文件開頭的官方 URL），但 `cloudbuild.yaml` 的 `deploy-staging` step 與 `scripts/deploy-cloud-run.sh` 都只認得 `_GEMINI_SECRET` / `_RUN_SERVICE_ACCOUNT` 這兩個 substitution，沒有任何路徑可以把 `CONTEXT_RAIL_GEMINI_MODEL` 傳進 `gcloud run deploy` 的 `--set-env-vars`。這次任務範圍刻意不改這兩個檔案（避免在 10/6 前又新增一段沒跑過的邏輯），改成部署完成、拿到 revision 之後，另外手動跑一次環境變數更新：

```sh
gcloud run services update context-rail-staging \
  --region="${REGION}" \
  --update-env-vars=CONTEXT_RAIL_GEMINI_MODEL=gemini-3.6-flash
```

**未實測**：這條指令的語法依 [Cloud Run services update 官方文件](https://cloud.google.com/sdk/gcloud/reference/run/services/update) 慣例寫的，10/6 當天第一次跑務必核對 `gcloud run services describe context-rail-staging --region="${REGION}" --format=yaml` 裡 `env` 區塊真的多了這個變數，不要假設成功。

**失敗處置**：`--update-env-vars` 會與既有環境變數合併（不是整批覆蓋），而 `--set-env-vars` 官方明寫「All existing environment variables will be removed first」。它會不會連 `--set-secrets` 掛的 `GEMINI_API_KEY` 一起清掉，官方文件沒有明講（env-vars 與 secrets 是兩組獨立 flag，推論不會，但未實測）——不要賭，務必用 `--update-env-vars`；如需確認，前後各跑一次 `gcloud run services describe <svc> --format="value(spec.template.spec.containers[0].env)"` 比對 `GEMINI_API_KEY` 的 secretKeyRef 是否還在。如果這一步忘記做，`internal/change/advisor.go` 仍會 fallback 到程式碼內建的 `gemini-2.0-flash` 預設值；由於該模型已 deprecated 但尚未到 shutdown date（2026-06-01 早於今天 2026-09-27——換言之官方公告的這個 shutdown date 已經過去，模型是否仍可呼叫、或呼叫是否已經失敗，本手冊沒有查證，10/6 當天第一次跑步驟 6 要特別注意回應內容或 `evaluation.observations` 裡的 fallback 訊息）。

### 步驟 6 — 驗證 Gemini 真的被呼叫到（腳本沒有的驗證，需手動跑）

`scripts/deploy-cloud-run.sh` 的 smoke check **不驗證 Gemini**，只驗證基本 API。程式碼裡有沒有走 Gemini 只能從實際回應內容看。

**這裡的 body 格式很容易寫錯，本節第一版手冊寫錯過**：`POST /v1/projects/{project}/changes` 是走 `internal/http/changes.go:43,77-83`（`create` handler），body decoder 開了 `DisallowUnknownFields()`（`internal/http/changes.go:167`），對應的 Go 型別是 `change.CreateRequest`（`internal/change/service.go:98-101`）：

```go
type Mutation struct {
    Actor  string `json:"actor"`
    Reason string `json:"reason"`
}
type CreateRequest struct {
    Mutation          // 展平：actor / reason 在頂層
    Request Request `json:"request"`  // 巢狀在 "request" 底下
}
```

`Request` 本身的欄位定義在 `internal/change/model.go:45-60`：是 `target_environment_id`（不是 `target_environment`），沒有頂層 `data_classification`——分類值要放進 `business_constraints`（`map[string]string`，`model.go:57`）。`actor` 可以放 body 頂層，也可以省略改用 `X-ContextRail-Actor` header 帶入（`internal/http/changes.go:171-173`：body 沒給就退回讀 header）。`reason` 是必填（`internal/change/service.go:189-191`：空字串會被 `CodeReasonRequired` 擋掉，回 422 不是 400），`request.title` 也必填。

正確的 body：

```sh
BASE="https://context-rail-staging-<hash>-<region>.a.run.app"
curl -fsS -i -X POST "${BASE}/v1/projects/order-operations-portal/changes" \
  -H 'Content-Type: application/json' \
  -d '{
    "actor": "william",
    "reason": "cloud-run runbook rehearsal smoke test",
    "request": {
      "title": "smoke test change",
      "objective": "verify gemini advisor wiring",
      "scope_included": ["advisor smoke test"],
      "scope_excluded": ["production changes"],
      "acceptance_criteria": [{"id": "AC-001", "text": "advisor_source is present in the response"}],
      "allowed_paths": ["docs/operations/CLOUD_RUN_DEPLOY_2026-10-06.zh-TW.md"],
      "forbidden_actions": ["deploy to production"],
      "target_environment_id": "staging",
      "business_constraints": {"data_classification": "internal"},
      "requested_by": "william"
    }
  }'
```

**這個 body 已經在本機用 `docker run` 起容器實測過**（沒有設定任何 `GEMINI_API_KEY`）：回應是 **HTTP 200**，`change.status` 為 `NEEDS_INPUT`（缺 `owner_summary` 與 `business_constraints.expected_monthly_volume`——這是預期中的行為，不影響 advisor 有沒有跑，`evaluateVersion` 在 `internal/change/service.go:465-472` 一定會呼叫 advisor，不管最後 readiness 是不是 `NEEDS_INPUT`），每個 option 的 `"advisor_source"` 都是 `"rule-advisor"`。

**預期**：10/6 部署後、`GEMINI_API_KEY` 真的透過 Secret Manager 掛進容器，且第 2 節步驟 5.5 的 `CONTEXT_RAIL_GEMINI_MODEL=gemini-3.6-flash` 也已經套用時，同一個 body 打下去，回應裡每個 option 的 `"advisor_source"` 欄位（型別定義在 `internal/change/model.go:115`）應該變成 **`"gemini:gemini-3.6-flash"`**——這個字串是 `Name()` 方法回傳的 `"gemini:" + advisor.Model`（`internal/change/advisor.go:140`）。若步驟 5.5 還沒跑過，`Model` 沒有另外設定 `CONTEXT_RAIL_GEMINI_MODEL` 時預設仍是 `"gemini-2.0-flash"`（`internal/change/advisor.go:134-136`，`NewGeminiAdvisor` 的預設值邏輯）——這個預設值已被 Google 官方列為 deprecated（見第 1 節、文件開頭），10/6 當天若看到 `"advisor_source"` 是 `"gemini:gemini-2.0-flash"`，代表步驟 5.5 忘了跑，不是預期結果。若設了其他 `CONTEXT_RAIL_GEMINI_MODEL` 值，這裡就會是對應的 `"gemini:<該 model>"`，不是固定字串，看 10/6 當天實際設的環境變數而定。

若 Gemini 呼叫失敗（key 錯、網路不通、額度用盡），會**靜默 fallback**，`evaluation.observations` 裡會多一條 `"advisor gemini:<model> unavailable (<err>); rule-based candidates used"`（`internal/change/service.go:467-471`）——這條訊息是唯一能分辨「fallback 了」與「本來就沒設 key」的地方，10/6 當天務必檢查這個欄位，不要只看 HTTP 200 就當作 Gemini 有生效。
**失敗處置**：如果一直 fallback，先確認 secret 掛載進容器的環境變數真的叫 `GEMINI_API_KEY`（`gcloud run services describe ... --format=yaml` 看 `env` 區塊），再確認 key 本身在 Google AI Studio 是 active 且沒有地區限制。

### 步驟 7 — 重跑 smoke（單獨驗證，不必重新部署）

```sh
scripts/deploy-cloud-run.sh --smoke "${BASE}"
```

對應 `scripts/deploy-cloud-run.sh:14-27`。

---

## 3. 這次演練發現的落差，以及 0927 的修法與靜態驗證

**原本的落差（0926 版本紀錄）**：

1. **`GEMINI_API_KEY` 沒有任何自動化路徑進 Cloud Run**——`cloudbuild.yaml` 的 `deploy-staging` step 完全沒有 `--set-secrets` / `--update-env-vars`，`scripts/deploy-cloud-run.sh` 也没有相關參數。`docs/operations/CLOUD_RUN_BASELINE.md:35` 只是文字說明「應該設成 secret」，程式碼從未真的做。
2. **Cloud Run 執行身份未指定** → 落到專案預設 Compute SA，權限範圍不明確、也拿不到 Secret Manager 存取權。
3. **`secretmanager.googleapis.com` 沒有被 enable**（`scripts/deploy-cloud-run.sh` 只 enable 三個 API）。

**0927 修法**：以上三點直接寫進 `cloudbuild.yaml`（`deploy-staging` step 新增 `_GEMINI_SECRET` / `_RUN_SERVICE_ACCOUNT` substitution，並在 enable API 清單加上 `secretmanager.googleapis.com`）與 `scripts/deploy-cloud-run.sh`（新增 `GEMINI_SECRET` / `SKIP_GEMINI_SECRET` / `RUN_SERVICE_ACCOUNT` 環境變數，同步 enable secretmanager API），不再是「手冊記錄額外手動指令」，而是腳本本身的行為。`scripts/bootstrap-w1.sh` 沒有建立 SA／secret 的邏輯（那仍是步驟 3 的手動前置作業），只在呼叫 `deploy-cloud-run.sh` 前加了一行提醒：`RUN_SERVICE_ACCOUNT` 沒 export 就會被下游腳本擋下來。

新增的參數規格：

| 名稱 | 預設值 | 關閉方式 | 缺值時的行為 |
| --- | --- | --- | --- |
| `cloudbuild.yaml` substitution `_RUN_SERVICE_ACCOUNT` / 腳本環境變數 `RUN_SERVICE_ACCOUNT` | 無（必填，declared default 是空字串） | 不適用——這是必填值，沒有「關閉」的概念，只有「還沒設」 | 空值時**清楚報錯並退出**（腳本層在呼叫 `gcloud builds submit` 之前就擋下；`cloudbuild.yaml` 的 `deploy-staging` step 自己也擋一次，供直接手動 `gcloud builds submit` 呼叫時使用），訊息附上怎麼設（短名稱，例如 `context-rail-run`）與會解析成的完整 email |
| `cloudbuild.yaml` substitution `_GEMINI_SECRET` / 腳本環境變數 `GEMINI_SECRET` | `gemini-api-key` | 設成空字串（`_GEMINI_SECRET=`），或腳本設 `SKIP_GEMINI_SECRET=1` | 空值時**不帶 `--set-secrets`**，印一行 NOTE 說明會 fallback 成 rule-advisor；不是靜默略過 |

**靜態驗證（0927，全部在本機跑，沒有碰真的 GCP／docker）**：

- `bash -n scripts/deploy-cloud-run.sh`、`bash -n scripts/bootstrap-w1.sh` — 通過。
- `shellcheck scripts/deploy-cloud-run.sh` — 只有一個既有的 SC2015（info 等級，第 37 行，跟這次改動無關）；新增的程式碼沒有新的 finding。`shellcheck scripts/bootstrap-w1.sh` — 乾淨。
- `cloudbuild.yaml` 裡兩個 `entrypoint: bash` 的 step 被抽出來，用實際的 Cloud Build 替換規則（先解析 `${_VAR}`，再把 `$$` 折成 `$`）還原成真正會執行的 bash，分別過 shellcheck：`resolve-digest` 乾淨；`deploy-staging` 有一個 `SC2054`（`--labels` 值裡的逗號被誤判成陣列分隔符，已加 `# shellcheck disable=SC2054` 註解排除，這是已知的 false positive，不是真的 bug）。
- `python3 -c 'import yaml,sys; yaml.safe_load(open("cloudbuild.yaml"))'` — YAML 語法通過。
- 寫了一個假 `gcloud` shim（攔截 `config set` / `services enable` / `artifacts ...` / `builds submit` / `run deploy` / `run services describe` / `run revisions describe`，全部只印出收到的參數，不打任何真的網路請求）放在 PATH 最前面，`builds submit` 會被導到一支小 Python 模擬器，模擬 Cloud Build 自己的 substitution 解析規則後，真的用 `bash -ceu` 跑 `cloudbuild.yaml` 裡的 `resolve-digest` / `deploy-staging` 兩個 step（`build` / `push` 這兩個 docker step 刻意跳過，不在這次驗證範圍內，image 建置本身沒有被改動）。分別跑了「預設」「`SKIP_GEMINI_SECRET=1`」「沒設 `RUN_SERVICE_ACCOUNT`」三種情境，組出的 `gcloud run deploy` 參數與錯誤訊息都符合預期（完整輸出見本次任務回報）。

這次修法仍然是**新增／收斂旗標**，不是重寫既有的 build/push/digest 邏輯——image 建置流程、region 預設值等其他行為維持不變。

## 4. 部署後要回填的證據

| 檔案 | 現況（宣告值） | 10/6 之後要填的真實值 |
| --- | --- | --- |
| `README.md:9` | `` `<deployed Cloud Run URL — filled in at submission>` `` | 真實 Cloud Run URL |
| `evidence/EB-008/README.md:28` | 「**No real deployment.** Every deployment record in evidence is declared.」 | 改成指向真實 revision / digest 的記錄，並附上 `scripts/record-promotion.sh` 實際輸出（該腳本本身這次沒有跑，它是給*受管理的 demo application*〔`order-operations-portal`〕記錄部署用的，不是給 ContextRail 自己） |
| `evidence/EB-009/README.md:35` | 同上一句（prod-demo 版本） | 同上，另外要有 prod-demo 服務自己的 revision/digest |
| `internal/change/advisor.go:124-125` 附近的程式碼註解「NOT exercised... treat as UNVERIFIED until a run with GEMINI_API_KEY is recorded as evidence」 | UNVERIFIED | 10/6 跑完步驟 6 後，把該次 HTTP 回應（`advisor_source: "gemini:..."`）存成新的 evidence bundle，並可以考慮把這句註解更新或移除 UNVERIFIED 標記（程式碼註解本身要不要改由 William 決定，這次沒有動它） |
| `decisions/DR-008-staging-promotion.json` / `DR-009-prod-demo-promotion.json` | `ACCEPTED_FOR_DEVELOPMENT` | 是否要升級決策狀態由 William 判斷，不在本次任務範圍內 |

## 5. 回滾方式

Cloud Run 對每次 `gcloud run deploy` 都會產生新 revision，不會覆蓋舊的（除非手動刪除），因此回滾是流量切回舊 revision：

```sh
gcloud run services describe context-rail-staging --region="${REGION}" \
  --format='value(status.traffic)'                      # 列出目前流量分配與 revision 清單
gcloud run services update-traffic context-rail-staging --region="${REGION}" \
  --to-revisions="<OLD_REVISION>=100"
```

**未實測**——`update-traffic` 語法依 GCP 官方文件；沒有在這次演練裡驗證過任何 revision 切換。若要完全撤回（例如誤設了錯誤的 secret 導致服務起不來），可以直接刪除新 revision：`gcloud run revisions delete <NEW_REVISION> --region="${REGION}"`，Cloud Run 會自動把流量留在僅存的 revision 上；同樣未實測。

## 6. 本次本地演練結果

環境：macOS `arm64`（Apple Silicon）、Docker Desktop 29.6.1、`gcloud` **未安裝**（`command not found: gcloud`，未嘗試安裝，因為任務範圍是唯讀 `--version` 或不裝）。

### Build

```sh
docker build --platform=linux/amd64 -t context-rail-local:rehearsal -f Dockerfile .
```

- **結果：成功**，總耗時約 45 秒（多為 npm ci 與 go mod download 的網路/快取時間；`go build` 步驟本身約 22 秒）。
- **image 大小：4.15 MB**（`docker image inspect` 回報 `size=4151887`，架構 `amd64`/`linux`）——distroless static base + 靜態連結的 Go binary（`CGO_ENABLED=0`，`-trimpath -ldflags="-s -w"`，見 `Dockerfile:23`）+ 前端 dist（一個 JS bundle 約 360KB + CSS 16KB，`frontend/dist` 總計約幾百 KB）+ 兩個 fixture 目錄（504KB + 60KB）。沒有觸發任何 Dockerfile 或 `.dockerignore` 問題。
  **澄清：4.15 MB 是 `docker images` / `docker inspect .Size` 回報的 CONTENT SIZE——也就是每層 gzip 壓縮後、實際會被 push 到 Artifact Registry 的傳輸大小**，不是容器實際跑起來時佔用的磁碟空間。用三種方式量了解壓後的實際大小：`docker history --no-trunc` 各層未壓縮大小加總 ≈ 14.3 MB；`docker export` 匯出容器的完整 rootfs（未壓縮 tar）≈ 11.55 MB；把該 tar 解開後 `du -sh`（macOS APFS，4K 區塊）≈ 13 MB。三個數字因量測方法不同而有差異，但都落在 **11–14 MB 這個量級**，不是 4.15 MB 那麼小；10/6 部署後 Cloud Run 的 overlay 檔案系統實際佔用量請以雲端環境自己的數字為準，不會剛好等於本地任何一個數字，但量級一致。
- 沒有跑 `docker build`（不加 `--platform`）比較 arm64/amd64 差異，因為沒有必要——Cloud Build 本來就在雲端 amd64 環境建置，本地是否用 `--platform` 只影響「這次演練的本地 tag」，不影響 10/6 實際部署路徑。

### Run + Smoke

```sh
docker run -d --name ctr-rehearsal --platform=linux/amd64 -p 18080:8080 \
  -e PORT=8080 \
  -e CONTEXT_RAIL_FIXTURE_ROOTS=/app/fixtures/order-operations-portal,/app/fixtures/support-insights \
  -e CONTEXT_RAIL_STATIC_DIR=/app/static \
  -e CONTEXT_RAIL_STATE_DIR=/tmp/context-rail-state \
  context-rail-local:rehearsal
```

四個環境變數其實與 Dockerfile 內建的 `ENV` 預設值完全相同（`Dockerfile:38-41`），這裡顯式帶入只是照任務指示做（真正需要覆寫的情境是換成 `CONTEXT_RAIL_FIXTURE_ROOTS=demo/order-operations-portal,examples/support-insights` 這種**相對路徑**，那是本機 `go run` 直接跑 binary 時用的寫法，不是容器內路徑；容器內固定是 `/app/fixtures/...`，两者不要混用）。

容器啟動 log：

```
ContextRail workspace listening on http://0.0.0.0:8080 (mode=container fixtures=[/app/fixtures/order-operations-portal /app/fixtures/support-insights] static=/app/static state=/tmp/context-rail-state advisor=rule-advisor read_back=none scenario=normal delay_ms=0)
```

curl 結果：

| 檢查 | 指令 | 結果 |
| --- | --- | --- |
| 健康檢查 | `curl http://127.0.0.1:18080/healthz` | `{"kind":"ContextRailHealth","status":"ok","mode":"container","surfaces":{...}}` |
| 前端靜態檔 | `curl -o /dev/null -w '%{http_code} %{content_type}' http://127.0.0.1:18080/` | `HTTP 200 content-type=text/html; charset=utf-8` |
| GET API | `curl http://127.0.0.1:18080/v1/projects` | 回傳兩個 fixture project（`order-operations-portal`、`support-insights`）的完整 registry snapshot |
| 唯讀邊界 | `curl -X POST http://127.0.0.1:18080/v1/projects` | `HTTP 405`（符合 `scripts/deploy-cloud-run.sh:21-22` 的 smoke 判斷邏輯） |

**這四項合起來就是 `scripts/deploy-cloud-run.sh` 裡 `smoke()` 函式（`scripts/deploy-cloud-run.sh:14-23`）在本地會做的完全相同的檢查**，只是這次打的是本地 port 18080 而不是 Cloud Run URL——這代表 smoke check 邏輯本身沒問題，10/6 部署到雲端後可以直接信任這支腳本。

跑完後：

```sh
docker stop ctr-rehearsal && docker rm ctr-rehearsal
```

**結果：容器已清除，沒有殘留**。

### 沒有跑的事（依任務範圍刻意留白）

- 沒有 `docker push`、沒有登入任何 registry、沒有跑任何 `gcloud` 寫入指令、沒有建立任何雲端資源。
- 沒有跑 `gcloud --version`（因為未安裝，`command not found`，已如實記錄，不用猜測版本）。
- 沒有驗證 Gemini 呼叫（本地沒有 `GEMINI_API_KEY`，容器 log 顯示 `advisor=rule-advisor`，這是預期行為，不是 bug）。
- 沒有跑 Playwright/E2E，只做了 curl 層級的手動 smoke（範圍內只要求健康檢查 + 一兩個 GET）。

## 7. 已知風險與未決項

1. **Secret Manager 路徑本身仍未實測**——0927 把 `--set-secrets` / `--service-account` 兩個旗標的**組裝邏輯**用假 `gcloud` shim 驗證過（第 3 節），但 secret 建立、IAM 綁定、以及 Cloud Run 真的把 secret 掛成環境變數這幾件事，都要 10/6 當天用真的 `gcloud` 才能驗——10/6 第一次跑務必保留每個指令的完整輸出，不要假設成功。
2. **`--min-instances=0`**（`cloudbuild.yaml:53`）代表 demo 展示中途若閒置過久，Cloud Run 可能把 instance 縮到零，`/tmp` 狀態（Change/Candidate/Release ledger）會全部遺失（`Dockerfile:34-37` 註解已明講）。10/6 展示前建議暫時改成 `--min-instances=1`（多花一點錢，換取展示期間不斷線），是否要改由 William 決定，本手冊只記錄風險，沒有代為修改 `cloudbuild.yaml`。
3. **`--service-account` 現在是必填，不會再靜默落到專案預設 Compute SA**——但如果 10/6 當天為了求快，把 `RUN_SERVICE_ACCOUNT` 設成專案預設 Compute SA 的名字（等於繞過第 2 節步驟 3 建專用 SA 的意義），要留意預設 SA 在較舊專案上可能有 Editor 角色（過寬），這不是這次任務能決定的政策問題，只記錄；旗標本身已經強制「一定要明確指定某個身份」，但不能阻止指定一個權限過寬的身份。
4. **`go.mod:3` 寫 `go 1.22`，但 `Dockerfile:17` 與 `.github/workflows/ci.yml:21,49` 都用 Go 1.24**——目前不影響建置（新版 toolchain 可建置宣告較舊 module），只是版本宣告不一致，記錄但不算部署風險。
5. **Gemini 是否真的被呼叫到，目前只能靠回應內容裡的 `advisor_source` 或 fallback 訊息字串**（第 2 節步驟 6）——沒有專門的健康檢查端點或 log 欄位直接標示「這次請求用了哪個 advisor」；10/6 當天做完整測試後，若要長期化，值得考慮加一個明確的 debug 端點或 log 行（本次任務未實作，超出「本地建置演練」範圍）。
6. **`scripts/record-promotion.sh` 是給*受管理的 demo application*（`order-operations-portal` 之類）記錄部署用的，不是給 ContextRail 自己的部署記錄**——手冊第 4 節已經標明這個區分，避免 10/6 當天搞混兩支腳本的用途。
7. **（0926 版本的決定，0927 已推翻）** 原本這份手冊刻意不動 `scripts/deploy-cloud-run.sh` / `cloudbuild.yaml` 本身，只記錄「額外跑的手動指令」。0927 由 William 決定改成直接把 `--set-secrets` / `--service-account` 寫進腳本與 `cloudbuild.yaml`（見第 3 節），理由是手動步驟在 10/6 當天容易漏做或打錯，寫進腳本可以讓「忘記設定」變成清楚報錯而不是靜默的安全缺口。代價是這兩個檔案本身沒有在真實 GCP 環境跑過（只有 shim dry-run，見第 3 節），10/6 當天如果 `gcloud run deploy` 报错，要同時排查「這次改動的旗標組裝邏輯」與「SA／secret 本身是否建好」兩條線。
8. **未實測（0928 補上）**：`--service-account` 這個旗標能不能成功套用，除了 runtime SA 本身要存在，還要求**執行 Cloud Build 的那個身份**對 runtime SA 有 `roles/iam.serviceAccountUser`（步驟 3.1）——這是本次任務新發現、之前手冊沒寫的缺口。目前只查了官方文件、寫了指令，**完全沒有在真實 GCP 專案跑過** `gcloud builds get-default-service-account` 或那條 `add-iam-policy-binding`；10/6 當天如果 `gcloud run deploy` 报 `PERMISSION_DENIED` 且訊息含 `iam.serviceAccounts.actAs`，先查這一步有沒有漏做，而不是先懷疑旗標組裝邏輯（旗標組裝已用 shim 驗證過，見第 3 節）。
9. **未實測（0927 補上）：程式碼預設的 Gemini model `gemini-2.0-flash` 已被 Google 官方列為 deprecated，官方公告的 shutdown date（2026-06-01）甚至早於今天（2026-09-27）**——來源：[Gemini API model deprecations](https://ai.google.dev/gemini-api/docs/deprecations)、[Gemini API models](https://ai.google.dev/gemini-api/docs/models)（官方建議替代 model：`gemini-3.6-flash`）。`cloudbuild.yaml` / `scripts/deploy-cloud-run.sh` 目前都沒有能傳遞 `CONTEXT_RAIL_GEMINI_MODEL` 的旗標（這次任務範圍刻意不改這兩個檔案），只在第 2 節步驟 5.5 補了部署後手動 `gcloud run services update --update-env-vars=CONTEXT_RAIL_GEMINI_MODEL=gemini-3.6-flash`，**這條指令本身完全沒有在真實 GCP 專案跑過**。風險有二：(a) 10/6 當天如果忘了跑步驟 5.5，容器會用已 deprecated 的預設值呼叫 Gemini，若該模型此時已真的被關閉，`GeminiAdvisor` 會呼叫失敗並靜默 fallback 成 rule-advisor（第 2 節步驟 6 已說明如何從 `evaluation.observations` 看出來），而不是清楚報錯；(b) `gemini-3.6-flash` 這個替代 model 名稱是本次任務用 `WebFetch` 讀官方頁面查到的，**沒有用真的 `GEMINI_API_KEY` 呼叫過這個 model id 驗證它確實可用**，10/6 當天第一次呼叫如果回應 404 或 model-not-found，要先假設是 model id 本身需要重新核對官方頁面（模型名稱與可用性會隨時間變動），不要假設是 IAM 或 secret 的問題。
10. **未實測（0927 補上）：`generationConfig.responseSchema` 已被官方標為 deprecated。** `internal/change/advisor.go` 用它約束 `pricing_ref` enum（DR-014）。官方 API reference（https://ai.google.dev/api/generate-content）逐字：「`responseSchema` (deprecated) … Use responseFormat instead.」；Structured Outputs 導覽頁（https://ai.google.dev/gemini-api/docs/structured-output）已改用 `v1beta/interactions` 的 `response_format`。欄位目前仍存在、棄用時程未公告。10/6 若 Gemini 回應被拒或 schema 被忽略：`EstimateOptionCost` 仍會獨立驗證 `pricing_ref`，清單外或缺漏一律 UNKNOWN；若 API 直接拒絕請求則 fallback 成 rule-advisor（清單內但與方案內容不符的風險仍在，見 DR-014 unknowns）；交件後再評估遷移到 `responseFormat`。
