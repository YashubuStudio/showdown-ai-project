import { buildLocalStudioFormat } from './showdown-suite-local-format';

export const Formats: import('../sim/dex-formats').FormatList = [
	{
		section: 'Showdown Suite',
	},
	buildLocalStudioFormat(),
];
