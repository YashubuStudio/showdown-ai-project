/**
 * Room-list panel (default right-panel)
 *
 * @author Guangcong Luo <guangcongluo@gmail.com>
 * @license AGPLv3
 */

import { PS, PSRoom, showdownSuiteLocalText, type RoomID, type RoomOptions } from "./client-main";
import { PSPanelWrapper, PSRoomPanel } from "./panels";
import type { RoomInfo } from "./panel-mainmenu";
import { toID } from "./battle-dex";

export class RoomsRoom extends PSRoom {
	override readonly classType: string = 'rooms';
	constructor(options: RoomOptions) {
		super(options);
		PS.send(`/cmd rooms`);
	}
}

type RoomsSection = [string, RoomInfo[]];
function localText(english: string, japanese: string) {
	return showdownSuiteLocalText(english, japanese);
}
class RoomsPanel extends PSRoomPanel {
	static readonly id = 'rooms';
	static readonly routes = ['rooms'];
	static readonly Model = RoomsRoom;
	static readonly location = 'right';
	static readonly icon = <i class="fa fa-plus rooms-plus" aria-hidden></i>;
	static readonly title = localText("Chat Rooms", "チャットルーム");
	hidden = false;
	search = '';
	section = '';
	lastKeyCode = 0;
	roomList: RoomsSection[] = [];
	roomListFocusIndex = -1;
	roomListLength = 0;
	override componentDidMount() {
		super.componentDidMount();
		this.subscriptions.push(PS.user.subscribe(update => {
			if (!update && PS.user.named) PS.send(`/cmd rooms`);
		}));
	}
	override componentDidUpdate() {
		const el = this.base?.querySelector('a.blocklink.cur');
		if (!this.roomListFocusIndex) return;
		el?.scrollIntoView({ behavior: 'auto', block: 'center' });
	}
	hide = (ev: Event) => {
		ev.stopImmediatePropagation();
		PS.hideRightRoom();
	};
	changeSearch = (ev: Event) => {
		const target = ev.currentTarget as HTMLInputElement;
		if (target.selectionStart !== target.selectionEnd) return;
		this.updateRoomList(target.value);
		this.forceUpdate();
	};
	changeSection = (ev: Event) => {
		const target = ev.currentTarget as HTMLSelectElement;
		this.section = target.value;
		this.forceUpdate();
	};
	handleOnBlur = (ev: Event) => {
		this.roomListFocusIndex = -1;
		this.forceUpdate();
	};
	keyDownSearch = (ev: KeyboardEvent) => {
		this.lastKeyCode = ev.keyCode;
		if (ev.shiftKey || ev.ctrlKey || ev.altKey || ev.metaKey) return;
		if (ev.keyCode === 38) { // up
			this.roomListFocusIndex = Math.max(this.roomListFocusIndex - 1, this.search ? 0 : -1);
			this.forceUpdate();
			ev.preventDefault();
		} else if (ev.keyCode === 40) { // down
			this.roomListFocusIndex = Math.min(this.roomListFocusIndex + 1, this.roomListLength - 1);
			this.forceUpdate();
			ev.preventDefault();
		} else if (ev.keyCode === 13) { // enter
			const target = ev.currentTarget as HTMLInputElement;
			let value = this.getRoomListFocusTitle() || target.value;
			const arrowIndex = value.indexOf(' \u21d2 ');
			if (arrowIndex >= 0) value = value.slice(arrowIndex + 3);
			if (!/^[a-z0-9-]$/.test(value)) value = toID(value);
			ev.preventDefault();
			ev.stopImmediatePropagation();
			target.value = '';
			this.updateRoomList('');

			PS.join(value as RoomID);
		}
	};
	updateRoomList(search?: string) {
		if (search) search = toID(search);
		const forceNoAutocomplete = this.search === `${search || ''}-`;
		if (search || this.search) {
			if (search === undefined || search === this.search) return;
			this.search = search;
			this.roomListFocusIndex = this.search ? 0 : -1;
		}
		this.roomList = this.getRoomList(forceNoAutocomplete);
		for (const [, rooms] of this.roomList) {
			rooms.sort((a, b) => (b.userCount || 0) - (a.userCount || 0));
		}
	}
	getRoomList(forceNoAutocomplete?: boolean): RoomsSection[] {
		const searchid = toID(this.search);

		if (!searchid) {
			const roomsCache = PS.mainmenu.roomsCache;
			let spotLightLabel = '';
			const officialRooms = [], chatRooms = [], hiddenRooms = [], spotLightRooms = [];
			for (const room of roomsCache.chat || []) {
				if (room.section !== this.section && this.section !== '') continue;
				if (room.privacy === 'hidden') {
					hiddenRooms.push(room);
				} else if (room.spotlight) {
					spotLightLabel = room.spotlight;
					spotLightRooms.push(room);
				} else if (room.section === 'Official') {
					officialRooms.push(room);
				} else {
					chatRooms.push(room);
				}
			}
			return [
				[localText("Official chat rooms", "公式チャット"), officialRooms],
				[spotLightLabel, spotLightRooms],
				[localText("Chat rooms", "チャットルーム"), chatRooms],
				[localText("Hidden rooms", "非表示ルーム"), hiddenRooms],
			];
		}

		let exactMatch = false;

		const rooms = PS.mainmenu.roomsCache;
		let roomList = [...(rooms.chat || [])];
		for (const room of roomList) {
			if (!room.subRooms) continue;
			for (const title of room.subRooms) {
				roomList.push({
					title,
					desc: `(Subroom of ${room.title})`,
				});
			}
		}

		let results = roomList.filter(room => {
			const titleid = toID(room.title);
			if (titleid === searchid) exactMatch = true;
			return titleid.startsWith(searchid) ||
				toID(room.title.replace(/^The /, '')).startsWith(searchid);
		});
		roomList = roomList.filter(room => !results.includes(room));

		results = results.concat(roomList.filter(room =>
			toID(room.title.toLowerCase().replace(/\b([a-z0-9])[a-z0-9]*\b/g, '$1')).startsWith(searchid) ||
			room.title.replace(/[^A-Z0-9]+/g, '').toLowerCase().startsWith(searchid)
		));

		const hidden: RoomsSection[] = !exactMatch ?
			[[localText("Possible secret room", "候補の秘密ルーム"), [{ title: this.search, desc: localText("(Private room?)", "（非公開ルームの可能性）") }]]] : [];

		const autoFill = this.lastKeyCode !== 127 && this.lastKeyCode >= 32;
		if (autoFill && !forceNoAutocomplete) {
			results.sort((a, b) => (b.userCount || 0) - (a.userCount || 0));
			const firstTitle = (results[0] || hidden[0][1][0]).title;
			let firstTitleOffset = 0;
			while (
				searchid !== toID(firstTitle.slice(0, firstTitleOffset)) &&
				firstTitleOffset < firstTitle.length // should never happen, but sanity against infinite loop
			) {
				firstTitleOffset++;
			}
			let autoFillValue = firstTitle.slice(firstTitleOffset);
			if (!autoFillValue && toID(firstTitle) !== searchid) {
				autoFillValue = ' \u21d2 ' + firstTitle;
			}
			const oldSearch = this.search;
			const searchElem = this.base!.querySelector<HTMLInputElement>('input[type=search]')!;
			if (autoFillValue) {
				searchElem.value = oldSearch + autoFillValue;
				searchElem.setSelectionRange(oldSearch.length, oldSearch.length + autoFillValue.length);
				this.search += '-';
			}

			return [[localText("Search results", "検索結果"), results], ...hidden];
		}

		return [...hidden, [localText("Search results", "検索結果"), results]];
	}
	override render() {
		if (this.hidden && PS.isVisible(this.props.room)) this.hidden = false;
		if (this.hidden) {
			return <PSPanelWrapper room={this.props.room} scrollable>{null}</PSPanelWrapper>;
		}
		const rooms = PS.mainmenu.roomsCache;
		this.updateRoomList();

		return <PSPanelWrapper room={this.props.room} scrollable><div class="pad">
			<button class="button" style="float:right;font-size:10pt;margin-top:3px" onClick={this.hide}>
				<i class="fa fa-caret-right" aria-hidden></i> {localText('Hide', '隠す')}
			</button>
			<div class="roomcounters">
				<a class="button" href="users" title={localText("Find an online user", "接続中のユーザーを探す")}>
					<span
						class="pixelated usercount"
						title="Meloetta is PS's mascot! The Aria forme is about using its voice, and represents our chatrooms."
					></span>
					<strong>{rooms.userCount || '-'}</strong> {localText('users online', '人が接続中')}
				</a> {}
				<a class="button" href="battles" title={localText("Watch an active battle", "進行中の対戦を観戦")}>
					<span
						class="pixelated battlecount"
						title="Meloetta is PS's mascot! The Pirouette forme is Fighting-type, and represents our battles."
					></span>
					<strong>{rooms.battleCount || '-'}</strong> {localText('active battles', '進行中の対戦')}
				</a>
			</div>
			<div>
				<select name="sections" class="button" onChange={this.changeSection}>
					<option value="">{localText('(All rooms)', '（すべての部屋）')}</option>
					{rooms.sectionTitles?.map(title => {
						return <option value={title}> {title} </option>;
					})}
				</select>
				<br /><br />
				<input
					type="search" name="roomsearch" class="textbox autofocus" style="width: 100%; max-width: 480px"
					placeholder={localText("Join or search for rooms", "部屋に参加 / 検索")} autocomplete="off"
					onInput={this.changeSearch} onKeyDown={this.keyDownSearch} onBlur={this.handleOnBlur}
				/>
			</div>
			{this.renderRoomList()}
		</div></PSPanelWrapper>;
	}
	getRoomListFocusTitle(): string | undefined {
		return this.roomList.map(([, rooms]) => rooms).reduce((a, b) => a.concat(b))[this.roomListFocusIndex]?.title;
	}
	renderRoomList() {
		const roomsCache = PS.mainmenu.roomsCache;
		if (roomsCache.userCount === undefined) {
			return <div class="roomlist"><h2>{localText('Official chat rooms', '公式チャット')}</h2><p><em>{localText('Connecting...', '接続中...')}</em></p></div>;
		}
		if (this.search) {
			// do nothing
		} else if (PS.isOffline) {
			return <div class="roomlist"><h2>{localText('Offline', 'オフライン')}</h2></div>;
		} else if (roomsCache.userCount === undefined) {
			return <div class="roomlist"><h2>{localText('Official chat rooms', '公式チャット')}</h2><p><em>{localText('Connecting...', '接続中...')}</em></p></div>;
		}

		// Descending order
		let nextOffset = 0;
		return this.roomList.map(([title, rooms]) => {
			if (!rooms.length) return null;

			const sortedRooms = rooms.sort((a, b) => (b.userCount || 0) - (a.userCount || 0));
			const offset = nextOffset;
			nextOffset += sortedRooms.length;
			this.roomListLength = nextOffset;

			const index = this.roomListFocusIndex >= offset && this.roomListFocusIndex < nextOffset ?
				this.roomListFocusIndex - offset : -1;

			return <div class="roomlist">
				<h2>{title}</h2>
				{sortedRooms.map((roomInfo, i) => <div key={roomInfo.title}>
					<a href={`/${toID(roomInfo.title)}`} class={`blocklink${i === index ? " cur" : ''}`}>
						{roomInfo.userCount !== undefined && <small style="float:right">({roomInfo.userCount} {localText('users', '人')})</small>}
						<strong><i class="fa fa-comment-o" aria-hidden></i> {roomInfo.title}<br /></strong>
						<small>{roomInfo.desc || ''}</small>
					</a>
					{roomInfo.subRooms && <div class="subrooms">
						<i class="fa fa-level-up fa-rotate-90" aria-hidden></i> {localText('Subrooms:', 'サブルーム:')} {}
						{roomInfo.subRooms.map(roomName => [<a href={`/${toID(roomName)}`} class="blocklink">
							<i class="fa fa-comment-o" aria-hidden></i> <strong>{roomName}</strong>
						</a>, ' '])}
					</div>}
				</div>)}
			</div>;
		});
	}
}

PS.addRoomType(RoomsPanel);
