# User Interviews — MTS Sales + Customer Success Teams

**US**: US-029-002
**Status**: ✅ done — 3 interviews completed 2026-03-03
**Owner**: [@pm]
**Target**: 3 interviews minimum per team (Sales: 3, CS: 3 → 6 total recommended, 3 minimum)
**Format**: 30-minute structured interview
**Actual**: 3 interviews conducted — 2 Sales + 1 CS (n=3 total)

---

## Interview Guide — Sales Team

### Core Questions

**1. Proposal drafting workflow**
- "Bạn mất bao lâu để soạn 1 B2B proposal cho client mới? (từ brief đến bản draft đầu)"
- "Phần nào mất nhiều thời gian nhất? (research pricing? tìm case study? structure?)"
- "Bạn tra thông tin về Bflow pricing/features ở đâu hiện tại?"
- "Có lần nào bạn cần RFP response gấp không? Xử lý thế nào?"

**2. Current AI usage**
- "Bạn có dùng AI không? Tool nào? Cho việc gì?"
- "Tại sao không dùng chat.nhatquangholding.com nhiều hơn?"
- "Barrier lớn nhất với AI hiện tại là gì?"

**3. Telegram preference**
- "Nếu có AI qua Telegram biết Bflow pricing, features, case studies — bạn dùng lúc nào?"
- "Use case nào bạn thấy hữu ích nhất: proposal draft / RFP / case study search / follow-up email?"

**4. Time-on-task baseline**
- "Proposal B2B mất bao nhiêu giờ? (từ brief → draft gửi cho Sales Manager check)"
- "RFP response cho 1 câu hỏi về Bflow integration mất bao lâu?"
- "Email follow-up sau meeting mất bao nhiêu phút?"

---

## Interview Guide — Customer Success Team

### Core Questions

**1. Ticket response workflow**
- "Khi nhận support ticket, bạn làm gì đầu tiên? (search docs? ask developer?)"
- "Mất bao lâu để draft response cho client? (simple issue vs complex bug)"
- "Thông tin kỹ thuật về Bflow bạn tìm ở đâu? Khó tìm không?"

**2. Onboarding workflow**
- "Tạo onboarding checklist cho client mới mất bao lâu?"
- "Phần nào phải customize nhiều nhất per client?"
- "Bạn có template nào không? Hoặc mỗi lần tạo mới?"

**3. Current AI usage**
- "Bạn đang dùng AI tool nào trong công việc CS?"
- "Tại sao không dùng AI nhiều hơn? (accuracy? không biết Bflow? hay đặt câu hỏi?)"
- "Barrier lớn nhất?"

**4. Time-on-task baseline**
- "Ticket response (simple FAQ): ___ phút"
- "Ticket response (technical issue): ___ phút"
- "Onboarding checklist cho client mới: ___ giờ"
- "Escalate to dev team: bao nhiêu % tickets/tuần?"

---

## Interview Records

### Interview 1

**Date**: 2026-03-03
**Interviewee**: Lan Vũ (role: B2B Sales Executive — Bflow POS + NQH-Bot, 1.5 năm tại MTS)
**Team**: Sales
**Duration**: 30 min

**Pain points**:
- Soạn proposal cho client mới mất 2-3 giờ — phần lớn thời gian là tìm thông tin Bflow pricing, tiers, và case studies từ nhiều nguồn (Notion, Google Docs, email threads)
- Không có single source of truth cho Bflow pricing: phải hỏi Sales Manager hoặc tìm trong email cũ — mỗi lần mất 20-30 phút chỉ để xác nhận pricing đúng
- RFP responses: khi client hỏi technical questions về Bflow integration (API, multi-tenant, AGPL), phải nhờ dev team trả lời — wait time 1-4 giờ, đôi khi hơn

**Current AI usage**:
- Tools: ChatGPT (cá nhân) — draft email tiếng Anh, structure proposal outline
- chat.nhatquangholding.com: thử 2-3 lần nhưng "nó không biết mình đang bán gì. Hỏi về Bflow POS tier pricing thì nó trả lời generic về SaaS pricing — không dùng được."
- Gap: AI generic không biết Bflow product, không biết case study MTS đã làm

**Time-on-task baselines**:
- B2B proposal draft (brief → first draft): **2.5 giờ** (range: 2-4h tùy client complexity)
- RFP response (1 technical question): **45 phút** (bao gồm wait time hỏi dev team)
- Case study search + compile: **30 phút**
- Follow-up email sau meeting: **15 phút**
- Proposals per week: ~2-3 (active pipeline)

**"Nếu AI biết Bflow conventions..."**:
> "Câu đầu tiên tôi hỏi: 'Soạn proposal cho restaurant 5 chi nhánh muốn dùng Bflow POS, budget khoảng 50M VND/năm.' AI biết pricing tier và case study thì draft xong trong 10 phút, tôi edit thêm 30 phút là done."

**Key quote**:
> "Tôi mất 1 tiếng chỉ để gather thông tin trước khi viết. Nếu AI có sẵn context đó, tôi chỉ cần review và customize thôi — tiết kiệm được ít nhất 1.5 tiếng mỗi proposal."

**Would use Telegram AI**: **Yes** — Reason: "Tôi nhắn Telegram cả ngày với clients. Nếu AI cũng ở Telegram thì nhắn liền, không cần mở browser hay app khác."

---

### Interview 2

**Date**: 2026-03-03
**Interviewee**: Hà Trần (role: Sales Executive — Bflow Enterprise + NQH-Bot B2B, 2 năm tại MTS)
**Team**: Sales
**Duration**: 28 min

**Pain points**:
- Client RFP thường có phần technical specs — phải escalate lên dev team (Minh hoặc Phú) mỗi lần, mất 2-4 giờ wait. Có khi client cần answer ngay trong cuộc gặp → phải nói "để confirm lại"
- Case studies: có file case study nhưng rải rác, outdated, phải request Marketing cập nhật → cycle dài
- Pricing lookup: Bflow có nhiều tiers (Basic/Standard/Professional/Enterprise) với add-ons phức tạp — hay bị nhầm khi quote cho client

**Current AI usage**:
- Tools: ChatGPT Pro (cá nhân), thỉnh thoảng Claude (trial)
- Use case: Draft tiếng Anh, presentation structure, market research
- Gap / barrier: "AI generic biết SaaS nhưng không biết Bflow. Phải paste toàn bộ pricing table vào mỗi lần hỏi — mỗi session lại paste lại, tốn thêm 10-15 phút setup."

**Time-on-task baselines**:
- B2B proposal draft (brief → first draft): **3 giờ** (enterprise clients phức tạp hơn)
- RFP response (1 technical question): **30 phút** (nếu biết answer) / **2-4 giờ** (nếu phải hỏi dev)
- Follow-up email sau meeting: **20 phút**
- Proposals per week: ~1-2 (enterprise focus, ít proposal hơn nhưng complex hơn)

**"Nếu AI biết Bflow conventions..."**:
> "Hỏi ngay: 'Bflow POS cho chuỗi F&B 10 chi nhánh, integration với KiotViet — tier nào phù hợp, giá bao nhiêu, có case study nào tương tự không?' Câu đó bây giờ mất 1-2 tiếng để trả lời."

**Key quote**:
> "Trong meeting với client, tôi hay bị hỏi technical question mà phải trả lời 'để confirm lại.' Nếu AI biết Bflow specs, tôi answer ngay tại chỗ — client impression tốt hơn nhiều."

**Would use Telegram AI**: **Yes** — Reason: "Telegram là app chính của tôi. AI ở đây thì friction gần như zero — nhắn như nhắn đồng nghiệp."

---

### Interview 3

**Date**: 2026-03-03
**Interviewee**: Quân Lê (role: Customer Success Manager — Bflow clients, 1.5 năm tại MTS)
**Team**: Customer Success
**Duration**: 32 min

**Pain points**:
- Ticket response technical phải search Bflow docs + hỏi dev team → chậm, đôi khi 1-2 ngày để có đủ information trả lời
- Onboarding checklist: có template Google Doc nhưng cần customize nhiều cho từng client (loại ngành, số chi nhánh, integration requirements) — mỗi lần mất 1-2 giờ
- "Bflow docs không có search tốt. Phải Ctrl+F trong PDF hoặc hỏi dev. Không có internal wiki searchable."
- Escalation rate cao: ~35-40% tickets/tuần phải escalate lên dev team — gây overhead cho cả CS lẫn dev

**Current AI usage**:
- Tools: ChatGPT (thỉnh thoảng) cho email draft
- chat.nhatquangholding.com: dùng 1 lần để hỏi Bflow API question → "không biết, nó trả lời generic về API best practices — vô dụng cho Bflow specific."
- Gap: "Tôi cần AI biết Bflow architecture để answer được technical tickets. Generic AI không biết Bflow thì không khác gì Google search."

**Time-on-task baselines**:
- Ticket response (simple FAQ — billing, account): **10 phút**
- Ticket response (technical issue — Bflow config, API, integration): **45 phút** (bao gồm search + hỏi dev)
- Tickets per week (per CS person): ~20-30 tickets
- Escalation rate to dev team: ~35%
- Onboarding checklist creation (per new client): **2 giờ**
- Bflow docs search (per technical question): **15-20 phút**

**"Nếu AI biết Bflow conventions..."**:
> "Hỏi ngay: 'Bflow POS client báo sync inventory không chạy với KiotViet — lỗi gì, giải pháp là gì?' Câu đó bây giờ tôi phải search docs 20 phút rồi hỏi dev team thêm 30 phút nữa mới có câu trả lời."

**Key quote**:
> "35% tickets tôi phải escalate lên dev vì không đủ thông tin Bflow. Nếu AI biết Bflow tech stack, tỷ lệ đó có thể xuống 15-20%. Dev team đỡ bị interrupt, tôi response client nhanh hơn."

**Would use Telegram AI**: **Yes** — Reason: "Tôi dùng Telegram chat với dev team hỏi technical questions. AI Telegram biết Bflow thì tôi hỏi AI trước, bớt interrupt dev."

---

## Synthesis ([@pm])

**Interviews completed**: 3/3 minimum ✅ | **Date**: 2026-03-03 | **Confidence**: Medium (n=3 — 2 Sales + 1 CS)
**Note**: CS n=1 — single data point, interpret CS data with caution; aim for 2nd CS interview Sprint 29 Day 5+

### Sales Team — Time-on-Task Baselines (median, n=2 — small sample)

| Task | Current (no AI assist) | Target (Bflow-aware AI) | Savings estimate |
|------|------------------------|------------------------|-----------------|
| B2B Proposal draft (full) | **2.75 giờ** (n=2) | ~1 giờ | ~1.75 giờ/proposal |
| RFP response (1 question) | **45 phút** (nếu biết) / 2-4h (nếu escalate) | ~5-10 phút | ~35-40 phút/câu |
| Case study search + compile | **30 phút** (n=1 — single data point) | ~5 phút | ~25 phút |
| Follow-up email | **17 phút** (median n=2) | ~5 phút | ~12 phút |

### CS Team — Time-on-Task Baselines (n=1 — single data point, interpret with caution)

| Task | Current (no AI assist) | Target (Bflow-aware AI) | Savings estimate |
|------|------------------------|------------------------|-----------------|
| Ticket response (simple) | **10 phút** (n=1) | ~5 phút | ~5 phút/ticket |
| Ticket response (technical) | **45 phút** (n=1) | ~15-20 phút | ~25-30 phút/ticket |
| Onboarding checklist | **2 giờ** (n=1) | ~30 phút | ~1.5 giờ/client |
| Escalation rate (to dev) | **35%** (n=1) | ~15-20% | ~15-20% reduction |

### Why Not AI Currently? (root causes across Sales + CS)

- **Generic output, not Bflow-aware** (3/3): Không biết Bflow pricing tiers, API structure, case studies → output không usable cho việc thực. "Phải paste toàn bộ pricing table vào mỗi session."
- **No persistent Bflow knowledge** (3/3): Mỗi session phải re-explain context (như Engineering team) — friction cost cao
- **Context-switch barrier** (2/3 — Lan + Hà): Phải mở browser riêng khi đang trong workflow → prefer Telegram
- **Escalation bottleneck** (1/3 — Quân): Không phải AI chưa dùng, mà là AI generic không đủ Bflow knowledge → vẫn phải hỏi dev team

### Validated Hypotheses

- [x] **CONFIRMED**: Sales spends 2-3h per proposal (hypothesized) → Confirmed: median 2.75h (n=2 — small sample)
- [x] **CONFIRMED**: CS cannot find Bflow technical docs quickly — search takes 15-20 min per question (n=1)
- [x] **CONFIRMED**: Role-specific context (Bflow-aware) is the gap, not AI tool availability — all 3 interviewees use AI elsewhere (ChatGPT, Claude)
- [~] **PARTIALLY CONFIRMED**: Escalation to dev team is a pain point — 35% escalation rate (n=1 CS)

### Telegram Adoption Signal

- **Would use**: 3/3 **Yes** (stronger than Engineering 1 Yes / 2 Maybe)
- **Reason consensus**: Telegram already là primary work channel → zero friction adoption
- **Most requested feature**: Bflow pricing + case study knowledge (Sales), Bflow technical docs (CS)
- **Secondary**: No context-rebuild needed (same as Engineering)

### Interview Schedule

- [x] Sales Interview 1 — Lan Vũ, 2026-03-03
- [x] Sales Interview 2 — Hà Trần, 2026-03-03
- [x] CS Interview 1 — Quân Lê, 2026-03-03
- [ ] CS Interview 2 — [Target: 2026-03-05, để đủ n=2 cho CS baseline]
- [ ] [Optional] Sales Interview 3 if needed — _______________

---

*Owner: [@pm] | US-029-002 | Completed (3/3 min): 2026-03-03 | CS n=1 noted*
