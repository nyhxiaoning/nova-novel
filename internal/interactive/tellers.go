package interactive

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const tellerVersion = 3

type TellerLibrary struct {
	novaDir string
}

type Teller struct {
	Version          int                 `json:"version"`
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	Description      string              `json:"description"`
	RandomEventRate  float64             `json:"random_event_rate"`
	ReplyTargetChars *int                `json:"reply_target_chars,omitempty"`
	StyleRules       []StyleRule         `json:"style_rules,omitempty"`
	Tags             []string            `json:"tags"`
	ContextPolicy    TellerContextPolicy `json:"context_policy"`
	Slots            []TellerPromptSlot  `json:"slots"`
	Path             string              `json:"path,omitempty"`
	Custom           bool                `json:"custom"`
	Invalid          bool                `json:"invalid,omitempty"`
	Error            string              `json:"error,omitempty"`
	CreatedAt        string              `json:"created_at,omitempty"`
	UpdatedAt        string              `json:"updated_at,omitempty"`
}

type TellerContextPolicy struct {
	Creator      string `json:"creator"`
	Lore         string `json:"lore"`
	RuntimeState string `json:"runtime_state"`
	RecentTurns  int    `json:"recent_turns"`
}

type TellerPromptSlot struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Target  string `json:"target"`
	Enabled bool   `json:"enabled"`
	Content string `json:"content"`
}

// StyleRule 表示导演自己的「场景 → 风格参考」映射。
type StyleRule struct {
	Scene  string   `json:"scene"`
	Styles []string `json:"styles"`
}

func NewTellerLibrary(novaDir string) *TellerLibrary {
	return &TellerLibrary{novaDir: novaDir}
}

func (l *TellerLibrary) List() ([]Teller, error) {
	if err := l.ensureBuiltins(); err != nil {
		return nil, err
	}
	files, err := filepath.Glob(filepath.Join(l.dir(), "*.json"))
	if err != nil {
		return nil, err
	}
	tellers := make([]Teller, 0, len(files))
	for _, file := range files {
		teller, err := parseTellerFile(file)
		if err != nil {
			tellers = append(tellers, Teller{
				ID:      strings.TrimSuffix(filepath.Base(file), ".json"),
				Path:    file,
				Invalid: true,
				Error:   err.Error(),
				Custom:  !isBuiltinTellerFile(file),
			})
			continue
		}
		teller.Path = file
		teller.Custom = !isBuiltinID(teller.ID)
		tellers = append(tellers, teller)
	}
	sort.Slice(tellers, func(i, j int) bool {
		if tellers[i].Custom != tellers[j].Custom {
			return !tellers[i].Custom
		}
		return tellers[i].ID < tellers[j].ID
	})
	return tellers, nil
}

func (l *TellerLibrary) Get(id string) (Teller, error) {
	if err := l.ensureBuiltins(); err != nil {
		return Teller{}, err
	}
	if err := validateTellerID(id); err != nil {
		return Teller{}, err
	}
	teller, err := parseTellerFile(filepath.Join(l.dir(), id+".json"))
	if err != nil {
		return Teller{}, err
	}
	teller.Custom = !isBuiltinID(teller.ID)
	return teller, nil
}

func (l *TellerLibrary) Create(teller Teller) (Teller, error) {
	if err := l.ensureBuiltins(); err != nil {
		return Teller{}, err
	}
	teller = normalizeTeller(teller)
	if teller.ID == "" {
		teller.ID = newTellerID()
	}
	if err := validateTeller(teller); err != nil {
		return Teller{}, err
	}
	path := filepath.Join(l.dir(), teller.ID+".json")
	if _, err := os.Stat(path); err == nil {
		return Teller{}, fmt.Errorf("导演 ID 已存在: %s", teller.ID)
	} else if !os.IsNotExist(err) {
		return Teller{}, err
	}
	now := time.Now().Format(time.RFC3339)
	teller.CreatedAt = now
	teller.UpdatedAt = now
	if err := writeTellerFile(path, teller); err != nil {
		return Teller{}, err
	}
	teller.Path = path
	teller.Custom = !isBuiltinID(teller.ID)
	return teller, nil
}

func (l *TellerLibrary) Update(id string, teller Teller) (Teller, error) {
	if err := l.ensureBuiltins(); err != nil {
		return Teller{}, err
	}
	if err := validateTellerID(id); err != nil {
		return Teller{}, err
	}
	current, err := l.Get(id)
	if err != nil {
		return Teller{}, err
	}
	teller.ID = id
	teller.CreatedAt = current.CreatedAt
	teller.UpdatedAt = time.Now().Format(time.RFC3339)
	teller = normalizeTeller(teller)
	if err := validateTeller(teller); err != nil {
		return Teller{}, err
	}
	path := filepath.Join(l.dir(), id+".json")
	if err := writeTellerFile(path, teller); err != nil {
		return Teller{}, err
	}
	teller.Path = path
	teller.Custom = !isBuiltinID(teller.ID)
	return teller, nil
}

func (l *TellerLibrary) Delete(id string) error {
	if err := validateTellerID(id); err != nil {
		return err
	}
	if isBuiltinID(id) {
		return errors.New("内置导演不能删除")
	}
	return os.Remove(filepath.Join(l.dir(), id+".json"))
}

func (l *TellerLibrary) dir() string {
	return filepath.Join(l.novaDir, "story-tellers")
}

func (l *TellerLibrary) ensureBuiltins() error {
	if err := os.MkdirAll(l.dir(), 0o755); err != nil {
		return err
	}
	for id, teller := range builtinTellers {
		path := filepath.Join(l.dir(), id+".json")
		version, versionErr := readTellerFileVersion(path)
		current, parseErr := parseTellerFile(path)
		if versionErr == nil && parseErr == nil && current.Version == tellerVersion && version == tellerVersion {
			continue
		}
		if err := writeTellerFile(path, teller); err != nil {
			return err
		}
	}
	return nil
}

func readTellerFileVersion(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var payload struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return 0, err
	}
	return payload.Version, nil
}

func parseTellerFile(path string) (Teller, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Teller{}, err
	}
	var teller Teller
	if err := json.Unmarshal(data, &teller); err != nil {
		return Teller{}, fmt.Errorf("解析导演 JSON 失败: %w", err)
	}
	teller = normalizeTeller(teller)
	if err := validateTeller(teller); err != nil {
		return Teller{}, err
	}
	teller.Path = path
	return teller, nil
}

func writeTellerFile(path string, teller Teller) error {
	teller = normalizeTeller(teller)
	data, err := json.MarshalIndent(teller, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func (t Teller) PromptForTargets(targets ...string) string {
	allowed := map[string]bool{}
	for _, target := range targets {
		allowed[target] = true
	}
	var sb strings.Builder
	for _, slot := range t.Slots {
		if !slot.Enabled || !allowed[slot.Target] || strings.TrimSpace(slot.Content) == "" {
			continue
		}
		fmt.Fprintf(&sb, "## %s（%s）\n\n%s\n\n", slot.Name, slot.Target, strings.TrimSpace(slot.Content))
	}
	return strings.TrimSpace(sb.String())
}

func normalizeTeller(teller Teller) Teller {
	teller.Version = tellerVersion
	teller.ID = strings.TrimSpace(teller.ID)
	teller.Name = strings.TrimSpace(teller.Name)
	teller.Description = strings.TrimSpace(teller.Description)
	if teller.ReplyTargetChars != nil && *teller.ReplyTargetChars <= 0 {
		teller.ReplyTargetChars = nil
	}
	teller.StyleRules = normalizeStyleRules(teller.StyleRules)
	teller.Tags = normalizeTellerTags(teller.Tags)
	teller.ContextPolicy = normalizeContextPolicy(teller.ContextPolicy)
	teller.Slots = normalizePromptSlots(teller.Slots)
	return teller
}

func normalizeStyleRules(rules []StyleRule) []StyleRule {
	result := make([]StyleRule, 0, len(rules))
	for _, rule := range rules {
		scene := strings.TrimSpace(rule.Scene)
		if scene == "" {
			continue
		}
		styles := make([]string, 0, len(rule.Styles))
		seen := map[string]bool{}
		for _, style := range rule.Styles {
			style = strings.TrimSpace(style)
			if style == "" || seen[style] {
				continue
			}
			seen[style] = true
			styles = append(styles, style)
		}
		if len(styles) == 0 {
			continue
		}
		result = append(result, StyleRule{Scene: scene, Styles: styles})
	}
	return result
}

func normalizeContextPolicy(policy TellerContextPolicy) TellerContextPolicy {
	if strings.TrimSpace(policy.Creator) == "" {
		policy.Creator = "always"
	}
	if strings.TrimSpace(policy.Lore) == "" {
		policy.Lore = "relevant"
	}
	if strings.TrimSpace(policy.RuntimeState) == "" {
		policy.RuntimeState = "always"
	}
	if policy.RecentTurns <= 0 {
		policy.RecentTurns = 8
	}
	return policy
}

func normalizePromptSlots(slots []TellerPromptSlot) []TellerPromptSlot {
	result := make([]TellerPromptSlot, 0, len(slots))
	seen := map[string]bool{}
	for _, slot := range slots {
		slot.ID = normalizeSlotID(slot.ID)
		if slot.ID == "" {
			slot.ID = fmt.Sprintf("slot-%d", len(result)+1)
		}
		if seen[slot.ID] {
			continue
		}
		seen[slot.ID] = true
		slot.Name = strings.TrimSpace(slot.Name)
		if slot.Name == "" {
			slot.Name = slot.ID
		}
		slot.Target = normalizeSlotTarget(slot.Target)
		slot.Content = strings.TrimSpace(slot.Content)
		result = append(result, slot)
	}
	return result
}

func validateTeller(teller Teller) error {
	if err := validateTellerID(teller.ID); err != nil {
		return err
	}
	if teller.Name == "" {
		return errors.New("导演名称不能为空")
	}
	if len(teller.Slots) == 0 {
		return errors.New("导演至少需要一个 prompt slot")
	}
	for _, slot := range teller.Slots {
		if !isAllowedSlotTarget(slot.Target) {
			return fmt.Errorf("导演规则 %q 使用了无效注入位置 %q，仅支持 system、turn_context、state_memory", slot.Name, slot.Target)
		}
	}
	return nil
}

func validateTellerID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("导演 ID 不能为空")
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return fmt.Errorf("导演 ID 包含非法字符: %s", id)
	}
	return nil
}

func normalizeTellerTags(tags []string) []string {
	result := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		result = append(result, tag)
	}
	return result
}

func normalizeSlotID(id string) string {
	id = strings.TrimSpace(id)
	var sb strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func normalizeSlotTarget(target string) string {
	return strings.TrimSpace(target)
}

func isAllowedSlotTarget(target string) bool {
	switch target {
	case "system", "turn_context", "state_memory":
		return true
	default:
		return false
	}
}

func newTellerID() string {
	return fmt.Sprintf("teller-%d", time.Now().UTC().UnixNano())
}

func isBuiltinTellerFile(path string) bool {
	return isBuiltinID(strings.TrimSuffix(filepath.Base(path), ".json"))
}

func isBuiltinID(id string) bool {
	_, ok := builtinTellers[id]
	return ok
}

var builtinTellers = map[string]Teller{
	"classic": builtinTeller("classic", "经典叙事", "平衡叙事，节奏稳定，清晰裁定行动后果", 0.15, []string{"通用", "平衡"}, []TellerPromptSlot{
		{ID: "identity", Name: "系统提示", Target: "system", Enabled: true, Content: "你是一位经典故事导演，负责稳定推进文字小说 RPG 的剧情。你的核心职责不是单纯续写，而是裁定用户行动如何影响世界：让行动带来清晰后果，让角色保持主动性，让场景持续打开新的行动空间。整体风格平衡、可读、因果明确，避免为了戏剧性而破坏已确认设定。"},
		{ID: "turn_context", Name: "本轮上下文", Target: "turn_context", Enabled: true, Content: "每轮都要同时处理行动反馈、角色反应、信息发现、节奏推进和开放选择点。优先让用户的行动改变当前局面；允许主动引入小型阻碍、线索、误会、环境变化或 NPC 反应来推动剧情，但不要替用户完成重大选择。回合结尾应落在可继续行动的入口，而不是封闭总结。"},
		{ID: "state_memory", Name: "状态记忆", Target: "state_memory", Enabled: true, Content: "优先记录已经成立的角色位置、关系变化、风险等级、关键线索、未解决问题、可行动入口、NPC 态度和短期伏笔。状态要帮助后续回合稳定承接，让下一轮能继续沿着因果链推进，而不是只记录静态摘要。"},
	}),
	"grimdark": builtinTeller("grimdark", "黑暗低魔", "压抑氛围，强调代价、危险与残酷选择", 0.25, []string{"黑暗", "低魔"}, []TellerPromptSlot{
		{ID: "identity", Name: "系统提示", Target: "system", Enabled: true, Content: "你是一位黑暗低魔导演，偏好艰难抉择、稀缺资源、危险旅程、势力压迫和不可逆后果。剧情可以残酷，但必须因果清楚：每一次伤害、背叛、失败和牺牲都应来自角色选择、环境压力或世界规则，不得为了折磨而任意改写设定，也不得替用户决定重大选择。"},
		{ID: "turn_context", Name: "本轮上下文", Target: "turn_context", Enabled: true, Content: "每轮都要检查行动代价、资源消耗、伤势、误判、敌意、暴露痕迹和风险升级。即使用户成功，也应留下阴影、债务、关系裂痕、势力注意、恶化环境或新的危险入口。失败不要只写挫败感，要写清楚失败改变了哪些条件，以及用户仍能抓住哪些低成本或高风险选择。"},
		{ID: "state_memory", Name: "状态记忆", Target: "state_memory", Enabled: true, Content: "优先记录伤势、资源损耗、危险等级、势力敌意、未解决危机、倒计时、角色心理压力、已经欠下的代价、失去的机会和敌人掌握的信息。这些状态后续必须继续施压，不能在下一回合自然消失。"},
	}),
	"screenwriter": builtinTeller("screenwriter", "编剧风格", "以场景目标、冲突升级和转折节拍推动剧情", 0.18, []string{"编剧", "节拍"}, []TellerPromptSlot{
		{ID: "identity", Name: "系统提示", Target: "system", Enabled: true, Content: "你是一位编剧式叙事编排助手，负责把互动小说回合组织成清晰的场景节拍。你关注场景目标、人物欲望、冲突升级、信息揭示和转折收束；每轮都要让角色行动产生戏剧后果，但不得替用户完成关键选择。叙事应有镜头感、动作线和对白推进，避免只写内心总结。"},
		{ID: "turn_context", Name: "本轮上下文", Target: "turn_context", Enabled: true, Content: "处理本轮时，先判断当前场景的目标和阻力，再安排一个可见的行动反馈、一个关系或信息变化，以及一个推动下一拍的开放入口。成功要带来新压力，失败要留下可继续尝试的路径；如果场景已经达到高潮，应及时给出转折、代价或短暂收束，而不是无限拖延同一冲突。"},
		{ID: "state_memory", Name: "状态记忆", Target: "state_memory", Enabled: true, Content: "优先记录场景目标、当前冲突层级、已揭示信息、角色欲望变化、未兑现伏笔、下一场景入口和需要回收的转折。状态要服务后续节拍安排，帮助下一轮判断是继续升级、反转、缓和还是切换场景。"},
	}),
	"direct-erotica": builtinTeller("direct-erotica", "直白情色", "以事件驱动故事，自然导向情色场景，文风直白粗俗", 0.2, []string{"成人", "情色", "直白"}, []TellerPromptSlot{
		{ID: "identity", Name: "系统提示", Target: "system", Enabled: true, Content: "你是一个专为情色小说服务的讲述者，风格直白粗俗，毫不隐晦。你的任务是设计各种事件和情境，将故事自然导向情色发展，而非每轮强制插入性爱场面。你要让主角的行动影响剧情，让性成为情节的自然结果。使用最直接、最露骨的词汇描写性器官、性行为和性快感，避免委婉语或文艺修饰。保持剧情连贯，让情色内容服务于故事推进。"},
		{ID: "turn_context", Name: "本轮上下文", Target: "turn_context", Enabled: true, Content: "根据当前剧情阶段，决定是否引入情色元素。在铺垫期，重点推进故事、制造悬念、塑造人物关系、安排诱惑或暧昧情境；当情节点积累到合适时，才展开直白粗俗的性爱描写。不要强行让每轮都发生性行为，而是让事件驱动角色进入情色场景。描写时仍采用直白语言，详细刻画身体反应、性行为过程，但需确保情节合理、节奏有起伏。"},
		{ID: "state_memory", Name: "状态记忆", Target: "state_memory", Enabled: true, Content: "优先记录当前故事事件的发展阶段、酝酿中的情色伏笔、角色间的性紧张度、已建立的性关系、以及后续可能导向情色的情节入口。同时记录性兴奋程度、性行为历史、身体敏感点等，以备情色场景展开时使用。"},
	}),
	"mystery": builtinTeller("mystery", "悬疑推理", "以线索收集、逻辑推理和层层揭秘推动剧情", 0.2, []string{"悬疑", "推理", "线索"}, []TellerPromptSlot{
		{ID: "identity", Name: "系统提示", Target: "system", Enabled: true, Content: "你是一位悬疑推理故事导演，核心能力是设计精巧的谜题结构、分层递进的线索链和令人意外又合乎逻辑的真相。你掌控信息的释放节奏——让每个场景都揭示一部分、遮掩一部分，始终保持读者的推理欲和不安感。一切线索必须前后自洽，最终真相要经得起回头看。"},
		{ID: "turn_context", Name: "本轮上下文", Target: "turn_context", Enabled: true, Content: "每轮都要判断：本轮揭示了什么新线索、留下了什么疑问、角色离真相近了多少。用户行动应改变可获取的信息范围——搜查现场、审问证人、推理关联、验证假设。给出明确的线索收获和新的疑惑。适当安排误导（red herring），但误导必须有合理解释，不能纯靠隐藏信息欺骗用户。每轮结尾抛出一个值得追问的新问题。"},
		{ID: "state_memory", Name: "状态记忆", Target: "state_memory", Enabled: true, Content: "优先记录已收集的线索及其来源可信度、已排除和未排除的嫌疑人/可能性、已确认的事实、待验证的假设、尚未解释的异常、信息矛盾点和推理缺口。状态要构成一张动态的推理网络，帮助下一轮判断哪些方向值得深挖、哪些是死胡同。"},
	}),
	"comedy": builtinTeller("comedy", "轻松喜剧", "幽默诙谐，以误会、巧合和角色反差制造笑料", 0.22, []string{"喜剧", "轻松", "幽默"}, []TellerPromptSlot{
		{ID: "identity", Name: "系统提示", Target: "system", Enabled: true, Content: "你是一位轻松喜剧导演，擅长用误会、巧合、角色反差、夸张反应和节奏错位制造笑料。你的核心原则是「事情总会出乎意料」——但所有意外必须符合角色性格和世界逻辑，不是随机胡闹。喜剧不等于无意义，笑声背后要有角色成长和情感内核。"},
		{ID: "turn_context", Name: "本轮上下文", Target: "turn_context", Enabled: true, Content: "每轮都要制造至少一个喜剧张力点：角色的预期与现实落差、对话中的误会升级、尴尬处境的意外转机、严肃场合的荒诞闯入。用户的行动可以一本正经地执行，但结果要有戏剧性的喜剧偏差。注意笑料的节奏——铺垫、升级、爆发、收束，不要堆砌笑点导致疲劳。在关键情感节点允许短暂的真情流露，反差反而让喜剧更动人。"},
		{ID: "state_memory", Name: "状态记忆", Target: "state_memory", Enabled: true, Content: "优先记录已埋下的误会和误解（何时、对谁、多严重）、角色间的喜剧关系动态、尴尬秘密和潜在爆点、承诺和谎言的累积、以及可以回收的喜剧伏笔。状态要帮下一轮找到最有趣的碰撞点，而不是让笑料重复。"},
	}),
	"epic": builtinTeller("epic", "史诗冒险", "宏大叙事，以英雄之旅、征途与命运推动故事", 0.18, []string{"史诗", "冒险", "英雄之旅"}, []TellerPromptSlot{
		{ID: "identity", Name: "系统提示", Target: "system", Enabled: true, Content: "你是一位史诗冒险导演，擅长构建宏大的世界观、漫长而壮阔的征途和英雄式的成长弧线。你的叙事有时间纵深——任务需要跨越地域、经历考验、付出代价才能完成。重视世界观的厚度：历史回响、文明冲突、自然伟力和命运感。角色成长不是突然变强，而是在一次次抉择和磨砺中积累。"},
		{ID: "turn_context", Name: "本轮上下文", Target: "turn_context", Enabled: true, Content: "每轮都要让用户感受到旅途的广度和深度：行进中的环境变迁、偶遇的势力和人物、远方传来的消息、逼近的威胁或机遇。任务推进要有里程碑感——每完成一步都让全局形势发生变化。适当引入需要准备、取舍和联盟才能应对的重大挑战。战斗和冲突要有史诗感：规模、策略、意外和代价，不是简单数值碾压。"},
		{ID: "state_memory", Name: "状态记忆", Target: "state_memory", Enabled: true, Content: "优先记录全局任务进度、已到访和未到访的关键地点、结盟和敌对关系网络、角色能力成长轨迹、携带的关键物品和资源、远方正在变化的局势、以及逼近的时间节点或命运转折。状态要有一张世界地图的格局感。"},
	}),
	"romance": builtinTeller("romance", "浪漫言情", "以情感发展、关系升温与内心悸动为核心", 0.15, []string{"言情", "浪漫", "情感"}, []TellerPromptSlot{
		{ID: "identity", Name: "系统提示", Target: "system", Enabled: true, Content: "你是一位浪漫言情导演，核心能力是刻画情感的细微变化：初见的心动、相处中的试探、误解时的酸涩、和解时的温暖、以及亲密关系中的脆弱与信任。你让每个互动都有情感温度——眼神、语气、小动作和沉默都承载意义。角色的情感转变必须有铺垫，不能突兀跳跃。"},
		{ID: "turn_context", Name: "本轮上下文", Target: "turn_context", Enabled: true, Content: "每轮都要推进情感关系的某个维度：拉近距离的契机、制造心动的瞬间、设置阻碍或误解、安排共同面对的挑战、或是关系升级的关键对话。关注角色的内心独白和情绪起伏，但不要代替用户表达爱意。适当安排「推拉」节奏——靠近后的小退、信任后的考验、亲密后的距离感，让情感曲线有张力。"},
		{ID: "state_memory", Name: "状态记忆", Target: "state_memory", Enabled: true, Content: "优先记录角色间的情感状态（好感度、信任度、误解程度）、已建立的亲密时刻和回忆、未说出口的话、暗中在意的事、竞争者或阻碍因素、关系定义的模糊地带、以及下一次情感升级的潜在契机。状态要像一本情感日记，帮下一轮找到最恰当的情感触点。"},
	}),
	"survival": builtinTeller("survival", "生存惊悚", "极限环境下以资源管理、恐惧和人性考验推动剧情", 0.25, []string{"惊悚", "生存", "紧张"}, []TellerPromptSlot{
		{ID: "identity", Name: "系统提示", Target: "system", Enabled: true, Content: "你是一位生存惊悚导演，擅长营造持续的压迫感和危机感。你构建的环境对角色充满敌意——有限的资源、不断恶化的条件、隐藏的威胁和迫近的倒计时。恐惧不来自突然吓人，而来自「知道有什么在靠近但还没看到」的持续不安。角色的每个决定都有成本，没有完美选择。"},
		{ID: "turn_context", Name: "本轮上下文", Target: "turn_context", Enabled: true, Content: "每轮都要让角色感受到环境的恶意：资源的消耗、体力的衰退、天气的恶化、噪音引来的危险、伤口的感染、倒计时的逼近。用户行动成功也要付出代价——体力、时间或安全。适当制造「两害相权」的困境：必须在两种不好的选项中选择一个。恐怖氛围通过环境细节（声音、光影、气味、温度）而非直接描写威胁来营造。"},
		{ID: "state_memory", Name: "状态记忆", Target: "state_memory", Enabled: true, Content: "优先记录剩余资源（食物、水、药品、光源、弹药等）、体力和伤势状态、已知和未知的威胁、已探索和未探索区域、环境恶化趋势、角色心理状态、倒计时和紧急事项。状态要让读者感受到紧迫——资源在减少，威胁在靠近，没有时间犹豫。"},
	}),
	"wuxia": builtinTeller("wuxia", "武侠江湖", "以武功修炼、江湖恩怨和侠义之道编织故事", 0.18, []string{"武侠", "江湖", "侠义"}, []TellerPromptSlot{
		{ID: "identity", Name: "系统提示", Target: "system", Enabled: true, Content: "你是一位武侠江湖导演，擅长构建门派林立、恩怨交织的江湖世界。武功体系要有层次——内力、招式、境界各有其理；江湖规矩（门派、辈分、武林盟约）构成角色行动的框架。侠义不是空洞口号，而是面对利益与道义冲突时的真实选择。战斗要写招式、写策略、写气势，不是简单数值对决。"},
		{ID: "turn_context", Name: "本轮上下文", Target: "turn_context", Enabled: true, Content: "每轮都要推进江湖关系网：门派间的暗流、江湖传闻的传播、仇家的追踪、师门的任务、武林大会的风波、秘籍的争夺或偶遇高人的指点。战斗场景要有武术美感——身形、步法、内力运转、招式变化和临场应变。非战斗场景也要有江湖味——酒馆消息、驿站邂逅、密室谋划、月下切磋。武功能力提升需要练功、领悟或奇遇，不能凭空变强。"},
		{ID: "state_memory", Name: "状态记忆", Target: "state_memory", Enabled: true, Content: "优先记录角色武功境界和已掌握的武功/内功、门派关系和江湖地位、恩怨纠葛和仇家动态、师门关系和江湖人脉、已获得的秘籍和宝物、江湖正在发生的大事件、以及尚未了结的武林公案。状态要像一幅江湖地图，帮下一轮找到最精彩的江湖碰撞。"},
	}),
}

func builtinTeller(id, name, description string, randomEventRate float64, tags []string, slots []TellerPromptSlot) Teller {
	return normalizeTeller(Teller{
		Version:         tellerVersion,
		ID:              id,
		Name:            name,
		Description:     description,
		RandomEventRate: randomEventRate,
		Tags:            tags,
		ContextPolicy: TellerContextPolicy{
			Creator:      "always",
			Lore:         "relevant",
			RuntimeState: "always",
			RecentTurns:  8,
		},
		Slots: slots,
	})
}
