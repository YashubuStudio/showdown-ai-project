/**
 * A panel displaying lists of commands and some basic informational resources
 * @author mia-pi-git
 */
import { PS, PSRoom, isShowdownSuiteLocalMode, showdownSuiteLocalText, type RoomOptions } from "./client-main";
import { PSPanelWrapper, PSRoomPanel } from "./panels";
import { toID } from "./battle-dex";
declare const BattleChatCommands: Record<string, string[]>;
const localText = (english: string, japanese: string) => showdownSuiteLocalText(english, japanese);

class ResourceRoom extends PSRoom {
	override readonly classType: string = 'resources';
	override readonly canConnect = false;

	constructor(options: RoomOptions) {
		super(options);
		this.title = localText('Resources', '情報');
	}
	override connect() {}
}

class ResourcePanel extends PSRoomPanel<ResourceRoom> {
	static readonly id = 'resources';
	static readonly routes = ['resources'];
	static readonly Model = ResourceRoom;
	static readonly icon = <i class="fa fa-question-circle" aria-hidden></i>;
	static readonly title = 'Resources';

	override state = { search: '' };
	override receiveLine() {}
	override render() {
		const { room } = this.props;
		if (isShowdownSuiteLocalMode()) {
			return <PSPanelWrapper room={room} scrollable>
				<div className="pad">
					<h2>ローカル情報</h2>
					<hr />
					<p>このクライアントはローカル LAN 専用です。公開 Pokémon Showdown や Smogon には接続しません。</p>
					<ul>
						<li>対戦開始: メインメニューで形式を選び、「対戦開始」を押します。</li>
						<li>部屋一覧: 「チャットルーム」にはローカルサーバー上の部屋だけが表示されます。</li>
						<li>名前変更: 右上の「名前を決める」から行います。</li>
						<li>AI 学習: 別 PC から `showtrain` コマンドでこのサーバーへ接続します。</li>
					</ul>
					<hr />
					<strong>コマンド一覧:</strong>
					<p><code>/help [command]</code> で各コマンドの説明を確認できます。</p>
					<label for="search">コマンド検索:</label>{' '}
					<input
						name="search"
						placeholder="検索"
						style={{ width: '25%' }}
						value={this.state.search}
						onChange={e => this.setState({ search: toID((e.target as any)?.value) })}
					/>
					<br />
					<span>{this.getCommandList()}</span>
				</div>
			</PSPanelWrapper>;
		}
		return <PSPanelWrapper room={room} scrollable>
			<div className="pad">
				<h2>PS! Informational Resources</h2>
				<hr />
				<p>
					PS! is a wide and varied site, with more facets than can be covered here easily.
					<br />
					While this page chiefly documents the ever-shifting set of commands available to PS! users,{' '}
					here are some useful resources for newcomers:
				</p>
				<ul>
					<li>
						<a href="https://www.smogon.com/forums/threads/3676132/">Beginner's Guide to Pokémon Showdown</a>
					</li>
					<li>
						<a href="https://www.smogon.com/dp/articles/intro_comp_pokemon">An introduction to competitive Pokémon</a>
					</li>
					<li>
						<a href="https://www.smogon.com/sm/articles/sm_tiers">What do 'OU', 'UU', etc mean?</a>
					</li>
					<li>
						<a href="https://www.smogon.com/dex/sv/formats/">What are the rules for each format?</a>
					</li>
					<li>
						<a href="https://www.smogon.com/sv/articles/clauses">What is 'Sleep Clause' and other clauses?</a>
					</li>
					<li>
						<a href="https://www.smogon.com/articles/getting-started">Next Steps for Competitive Battling</a>
					</li>
					<li>
						<button className="button" data-cmd="/report">Report a user</button>
					</li>
					<li>
						<button className="button" data-cmd="/join help">Join the Help room for live help</button>
					</li>
				</ul>
				<hr />
				<strong>Commands:</strong>
				<p>
					Within any of the chats, and in private messages,{' '}
					it is possible to type in commands (messages beginning with <code>/</code>){' '}
					to perform a particular action. A great number of these commands exist, {' '}
					with some only available to certain users. For instance, you can broadcast commands to others with the{' '}
					<code>!</code> prefix, but only when you're a player in a battle or a Voice (+) user.
					<br />
					For more information on ranks, type <code>/groups</code> in any chat.{' '}
					You can also use the "chat self" button on your username in the top right {' '}
					if you need a place to send these commands without joining a room.
				</p>

				<details className="readmore">
					<summary>Here's a list of the most useful commands for the average Pokémon Showdown experience:</summary>
					<p>
						COMMANDS: /report, /msg, /reply, /logout,{' '}
						/challenge, /search, /rating, /whois, /user, /join, /leave, /userauth, /roomauth
					</p>
					<p>
						BATTLE ROOM COMMANDS: /savereplay, /hideroom, /inviteonly, /invite,{' '}
						/timer, /forfeit
					</p>
					<p>
						OPTION COMMANDS: /nick, /avatar, /ignore, /status, /away, /busy, /back, /timestamps,{' '}
						/highlight, /showjoins, /hidejoins, /blockchallenges, /blockpms
					</p>
					<p>
						INFORMATIONAL/RESOURCE COMMANDS: /groups, /faq, /rules, /intro, /formatshelp,{' '}
						/othermetas, /analysis, /punishments, /calc, /git, /cap, /roomhelp, /roomfaq{' '}
						(replace / with ! to broadcast. Broadcasting requires: + % @ # ~)
					</p>
					<p>
						DATA COMMANDS: /data, /dexsearch, /movesearch, /itemsearch, /learn,{' '}
						/statcalc, /effectiveness, /weakness, /coverage, /randommove, /randompokemon</p>
					<p>For an overview of room commands, use <code>/roomhelp</code></p>
					<p>For details of a specific command, you can use <code>/help [command]</code>, for example <code>/help data</code>.</p>
				</details>

				<br />
				<p>
					A complete list of commands available to regular users is provided below. Use {' '}
					<code>/help [commandname]</code> for more in-depth information on how to use them.
				</p>
				<hr />
				<label for="search">Search/filter commands:</label>{' '}
				<input
					name="search"
					placeholder="search"
					style={{ width: '25%' }}
					value={this.state.search}
					onChange={e => this.setState({ search: toID((e.target as any)?.value) })}
				/>
				<br />
				<span>{this.getCommandList()}</span>
			</div>
		</PSPanelWrapper>;
	}
	getCommandList() {
		if (typeof BattleChatCommands !== 'object') {
			document.addEventListener('ready', () => this.setState({ commandsLoaded: true }));
			return <>{localText('Loading command data, please try again in a few moments...', 'コマンド一覧を読み込み中です。少し待ってからもう一度開いてください...')}</>;
		}
		const buf = [];
		const search = this.state.search;
		const keys = Object.keys(BattleChatCommands).sort((a, b) => {
			// pin info to the top ALWAYS
			if (b.endsWith('info')) return 2;
			// prefer default commands near the top, more generally useful
			if (b.includes('chat-commands')) return 1;
			const aCount = BattleChatCommands[a].filter(x => toID(x).includes(search)).length;
			const bCount = BattleChatCommands[b].filter(x => toID(x).includes(search)).length;

			return (bCount - aCount) || a.localeCompare(b);
		}).filter(plugin => !(plugin.endsWith('admin')));

		for (let pluginName of keys) {
			const cmdTable = BattleChatCommands[pluginName];
			if (!cmdTable.length) continue;
			if (pluginName.startsWith('chat-plugins')) {
				pluginName = 'Chat plugin: ' + pluginName.split('/').slice(1).join('/');
			} else if (pluginName.startsWith('chat-commands')) {
				pluginName = 'Core commands: ' + pluginName.split('/').slice(1).join('/'); ;
			}
			let matchedCmds = [];
			for (const cmd of cmdTable) {
				if (search.length && !toID(cmd).includes(search)) {
					continue;
				}
				matchedCmds.push(<li>{cmd}</li>);
			}

			buf.push(
				<details class="readmore">
					<summary>
						<strong>{pluginName} {search.length ? <>({matchedCmds.length}))</> : ''}</strong>
					</summary>
					<ul>{matchedCmds}</ul>
				</details>
			);
			buf.push(<br />);
		}
		return buf;
	}
}

PS.addRoomType(ResourcePanel);
