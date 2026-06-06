package testutil

import "github.com/AsperforMias/ScriptForge/backend/internal/job"

func GrowthFantasyRealInputRequest() job.CreateJobRequest {
	var req job.CreateJobRequest
	req.Source.Title = "厄洛斯的转生见闻"
	req.Source.Author = "自定义输入作者"
	req.Adaptation.Style = "异世界转生 / 贵族成长"
	req.Adaptation.Audience = "青年向"
	req.Adaptation.Notes = []string{
		"优先保证 deterministic 输出可信，而不是补足完整题材能力。",
		"把叙述段落压成可拍场景初稿，避免把长从句直接塞进 objective、dialogue 与 open_questions。",
	}
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{
			Index:   1,
			Title:   "第一章 穿越之后",
			Content: "房门打开后，一个男人晃晃悠悠地倒在床上，因为酒劲和记忆回流一起昏睡过去。再次睁眼时，四周已经变成陌生的贵族府邸，金发碧眼的佣人和侍从把他当成小少爷厄洛斯。接下来的几年里，他确认自己不是短暂做梦，而是真的以公爵继承人的身份在这个世界重新长大。三岁那年，他就因为算术天赋被整个公国记住；五岁以后，他已经能在家族议事时听懂大人真正担心的是什么。厄洛斯很清楚，这一世不能只当被宠着的孩子，他得尽快站稳继承人的位置。",
		},
		{
			Index:   2,
			Title:   "第二章 书页里的旧纪元",
			Content: "厄洛斯在藏书馆翻看旧历史时，第一次从书页里看到迷雾海、精灵族和旧纪元崩塌的记载。比起只做吃喝不愁的公爵继承人，他更想查清这个世界真正被谁删掉了一部分历史。\n\n他把那几页内容重新抄在纸上，甚至开始怀疑家族收藏的禁书和北境边境的异象是否有关。厄洛斯意识到，这条神秘线索迟早会把自己拖进更大的局里。\n\n抱着书走出藏书馆时，他迎面碰上了正在找人的母亲艾丝黛儿。艾丝黛儿先问他是不是又躲开礼仪课，接着又追问温蒂尼究竟跑到哪里去了。\n\n等艾丝黛儿转身离开后，温蒂尼果然从花坛后面扑出来，一边骑到厄洛斯身上一边质问他是不是又准备向妈妈告状。厄洛斯一边安抚她，一边更明确地意识到，自己若想继续追查神秘侧，就得先把公爵府内部这层关系处理好。",
		},
		{
			Index:   3,
			Title:   "第三章 先把脚下的位置站稳",
			Content: "随着年纪渐长，厄洛斯开始更认真地看待自己作为继承人的身份。他在账房里翻看名册和税册，发现领地的税期、庄园收支和粮仓账目都没有表面上那么稳。越看下去，他越确定这不是一两笔疏漏能解释的账。\n\n他顺着账目缺口继续追查，发现北境几个庄园的欠税被人故意往后拖，真正要爆雷的时间点恰好会落到自己开始接触家族事务之后。厄洛斯明白，这不是抄两本账就能解决的事，也不是装作没看见就能拖过去的麻烦。\n\n之后他又被带去议事厅旁听，亲耳听见大人们在争论修渠、粮仓和边境治安的问题。那些原本只存在于纸面的数字，突然都变成了迟早会压到他头上的责任。会议里每个人都在推诿先后顺序，却没人真的愿意接下最难的部分。\n\n厄洛斯最终下定决心，要继续追查这个世界更深的秘密，前提是先把贵族继承人的位置站稳，把家族和领地的秩序推进到自己能掌控的状态。只要脚下的位置没站稳，任何更远的真相都会先把他拖垮。",
		},
	}
	return req
}

func GrowthFantasyRealInputExpectedNames() []string {
	return []string{"厄洛斯", "温蒂尼", "艾丝黛儿"}
}

func GrowthFantasyRealInputForbiddenFragments() []string {
	return []string{"脑子里", "三岁时", "因为听"}
}
