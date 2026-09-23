import { useId, useState } from 'react';
import {
  Apple,
  ArrowRight,
  Check,
  Clock3,
  Leaf,
  List,
  LockKeyhole,
  MessageSquare,
  Sprout,
  TreeDeciduous,
  Zap,
} from 'lucide-react';
import type { Experience, ModuleView } from '../api/types';
import { useI18n, useLabels } from '../i18n';
import { Empty, Status } from './Shared';
const branchPositions: Record<string, number[][]> = {
  professional: [
    [27, 27],
    [18, 43],
    [33, 53],
    [33, 69],
  ],
  personal: [
    [50, 13],
    [48, 31],
    [51, 48],
    [50, 67],
  ],
  people: [
    [72, 26],
    [82, 44],
    [67, 56],
    [69, 71],
  ],
};
export function TreeArt({
  id = 'tree',
  level = 1,
  children,
}: {
  id?: string;
  level?: number;
  children?: React.ReactNode;
}) {
  return (
    <svg viewBox="0 0 520 440" aria-hidden="true">
      <defs>
        <radialGradient id={`${id}-halo`}>
          <stop stopColor="#e4f0cf" />
          <stop offset="1" stopColor="#f5f9ed" stopOpacity="0" />
        </radialGradient>
        <linearGradient id={`${id}-trunk`} x2="1" y2="0">
          <stop stopColor="#779155" />
          <stop offset="1" stopColor="#a5bb7d" />
        </linearGradient>
      </defs>
      <ellipse cx="260" cy="208" rx="235" ry="201" fill={`url(#${id}-halo)`} />
      <ellipse className="tree-ground" cx="260" cy="391" rx="106" ry="12" />
      <g
        className="tree-canopy"
        transform={`translate(260 392) scale(${0.94 + Math.min(Math.max(level - 1, 0), 8) * 0.012}) translate(-260 -392)`}
      >
        <path
          d="M242 392 Q250 335 244 286 Q236 254 213 223 L225 215 Q247 241 254 263 Q264 196 257 155 L270 154 Q277 219 270 271 Q291 237 321 221 L327 231 Q285 261 274 307 Q270 350 280 392Z"
          fill={`url(#${id}-trunk)`}
        />
        <g className="tree-branch">
          <path data-branch="professional" d="M257 291 Q213 287 181 254 Q153 226 107 213" />
          <path data-branch="people" d="M264 270 Q305 282 345 253 Q374 234 418 214" />
          <path data-branch="professional" d="M255 233 Q211 220 186 177 Q163 143 145 116" />
          <path data-branch="people" d="M268 221 Q313 192 334 158 Q352 132 369 116" />
          <path data-branch="personal" d="M264 182 Q254 121 264 71" />
          <path
            className="tree-thin"
            d="M195 274 Q154 286 139 307 M172 248 L149 190 M326 265 Q362 280 381 304 M346 251 L377 180 M223 219 L214 157 M299 200 L308 124"
          />
        </g>
        <g>
          <path className="tree-leaf" d="M242 108 Q203 82 228 48 Q267 64 242 108Z" />
          <path className="tree-leaf light" d="M273 95 Q269 51 301 45 Q315 79 273 95Z" />
          <path className="tree-leaf" d="M195 177 Q148 177 151 139 Q190 139 195 177Z" />
          <path className="tree-leaf dark" d="M158 139 Q117 122 125 90 Q165 98 158 139Z" />
          <path className="tree-leaf light" d="M191 162 Q184 122 215 110 Q235 145 191 162Z" />
          <path className="tree-leaf light" d="M336 160 Q328 123 357 103 Q382 131 336 160Z" />
          <path className="tree-leaf" d="M354 139 Q365 91 398 103 Q400 139 354 139Z" />
          <path className="tree-leaf" d="M135 229 Q96 241 77 213 Q108 186 135 229Z" />
          <path className="tree-leaf light" d="M152 215 Q115 184 139 156 Q175 175 152 215Z" />
          <path className="tree-leaf dark" d="M130 254 Q91 268 80 240 Q111 220 130 254Z" />
          <path className="tree-leaf light" d="M395 225 Q400 181 432 190 Q440 226 395 225Z" />
          <path className="tree-leaf" d="M389 251 Q420 223 443 249 Q425 280 389 251Z" />
          <path className="tree-leaf" d="M219 241 Q184 227 194 196 Q231 202 219 241Z" />
          <path className="tree-leaf light" d="M284 250 Q285 205 318 211 Q321 245 284 250Z" />
          <path className="tree-leaf dark" d="M154 297 Q119 279 112 310 Q139 329 154 297Z" />
          <path className="tree-leaf light" d="M179 291 Q146 277 151 255 Q188 259 179 291Z" />
          <path className="tree-leaf" d="M355 289 Q355 255 386 267 Q397 297 355 289Z" />
          <path className="tree-leaf light" d="M376 305 Q390 281 413 302 Q405 330 376 305Z" />
          <path className="tree-leaf light" d="M226 334 Q195 310 179 341 Q205 360 226 334Z" />
          <path className="tree-leaf" d="M281 338 Q302 309 325 332 Q311 360 281 338Z" />
          <path
            d="M203 391q-8-19-20-13m107 12q17-21 24-15m-114 13q-1-12-8-14m111 13q-1-12 9-17"
            fill="none"
            stroke="#95ad70"
            strokeWidth="3"
            strokeLinecap="round"
          />
        </g>
        {children}
      </g>
    </svg>
  );
}
export default function DevelopmentTree({
  modules,
  experience,
  onOpen,
  celebrate = false,
  celebrateBranch = '',
}: {
  modules: ModuleView[];
  experience: Experience;
  onOpen: (module: ModuleView) => void;
  celebrate?: boolean;
  celebrateBranch?: string;
}) {
  const { tr } = useI18n();
  const labels = useLabels();
  const id = useId().replace(/:/g, '');
  const [view, setView] = useState<'tree' | 'list'>('tree');
  const [filter, setFilter] = useState('');
  const [page, setPage] = useState(0);
  const categories = [...new Set(modules.map((m) => m.category || m.branch))]
    .filter(Boolean)
    .sort();
  const filtered = modules.filter((m) => !filter || (m.category || m.branch) === filter);
  const groups = ['professional', 'personal', 'people'].map((branch) => ({
    branch,
    items: filtered
      .filter((m) => (m.branch || 'professional') === branch)
      .sort((a, b) => a.event.event_id.localeCompare(b.event.event_id)),
  }));
  const pageCount = Math.max(1, ...groups.map((group) => Math.ceil(group.items.length / 4)));
  const shown = groups.flatMap((group) =>
    group.items
      .slice(page * 4, page * 4 + 4)
      .map((module, index) => ({ module, position: branchPositions[group.branch][index] })),
  );
  const changeFilter = (value: string) => {
    setFilter(value);
    setPage(0);
  };
  return (
    <section
      className={`tree-panel panel ${celebrate ? 'tree-celebrate' : ''}`}
      data-celebrate-branch={celebrateBranch}
      aria-label={tr('Дерево развития', 'Даму ағашы', 'Development tree')}
    >
      <div className="section-heading">
        <div>
          <h2>{tr('Ваше дерево роста', 'Сіздің өсу ағашыңыз', 'Your growth tree')}</h2>
          <span className="tree-level">
            <Sprout size={12} />
            {tr('Уровень дерева', 'Ағаш деңгейі', 'Tree level')} {experience.tree_level}
          </span>
        </div>
        <div
          className="view-switch"
          role="group"
          aria-label={tr('Вид модулей', 'Модульдер көрінісі', 'Module view')}
        >
          <button
            className={view === 'tree' ? 'active' : ''}
            aria-pressed={view === 'tree'}
            onClick={() => setView('tree')}
          >
            <TreeDeciduous size={15} />
            {tr('Дерево', 'Ағаш', 'Tree')}
          </button>
          <button
            className={view === 'list' ? 'active' : ''}
            aria-pressed={view === 'list'}
            onClick={() => setView('list')}
          >
            <List size={15} />
            {tr('Список', 'Тізім', 'List')}
          </button>
        </div>
      </div>
      {view === 'tree' ? (
        <>
          <div className="tree-scene">
            <TreeArt id={id} level={experience.tree_level} />
            {shown.map(({ module: m, position: pos }, index) => (
              <button
                key={m.event.event_id}
                className={`tree-apple ${m.state} ${m.recommended ? 'recommended' : ''}`}
                data-branch={m.branch}
                style={{ left: `${pos[0]}%`, top: `${pos[1]}%` }}
                aria-label={`${m.event.title}. ${labels.category(m.category)}. ${labels.status(m.state)}. ${m.config.reward_exp} EXP`}
                onClick={() => onOpen(m)}
              >
                <Apple size={36} />
                <span className="apple-mark">
                  {m.state === 'completed' ? (
                    <Check />
                  ) : m.state === 'pending' ? (
                    <Clock3 />
                  ) : m.state === 'locked' ? (
                    <LockKeyhole />
                  ) : m.state === 'changes_requested' ? (
                    <MessageSquare />
                  ) : m.recommended ? (
                    <Zap />
                  ) : (
                    index + 1
                  )}
                </span>
                <span className="tree-tooltip">
                  {m.event.title}
                  <br />
                  {labels.status(m.state)}
                </span>
              </button>
            ))}
          </div>
          <div className="branch-labels">
            <span>{tr('Профессия', 'Мамандық', 'Professional')}</span>
            <span>{tr('Саморазвитие', 'Өзін-өзі дамыту', 'Personal')}</span>
            <span>{tr('Команда', 'Команда', 'People')}</span>
          </div>
          <div className="tree-legend">
            <span>
              <i className="legend-dot outline" />
              {tr('Модуль', 'Модуль', 'Module')}
            </span>
            <span>
              <i className="legend-dot" />
              {tr('Следующий шаг', 'Келесі қадам', 'Next step')}
            </span>
            <span>
              <i className="legend-dot gold" />
              {tr('Подтверждён', 'Расталған', 'Approved')}
            </span>
          </div>
        </>
      ) : (
        <div className="tree-list">
          {filtered.map((m) => (
            <button className="tree-list-item" key={m.event.event_id} onClick={() => onOpen(m)}>
              <Leaf size={19} />
              <span>
                <strong>{m.event.title}</strong>
                <small>
                  {labels.category(m.category || m.branch)} · {m.config.reward_exp} EXP
                </small>
              </span>
              <Status value={m.state} />
              <ArrowRight size={15} />
            </button>
          ))}
          {!filtered.length && (
            <Empty
              title={tr('Модулей пока нет', 'Модульдер әзірге жоқ', 'No modules yet')}
              text={tr(
                'Попробуйте другое направление.',
                'Басқа бағытты таңдаңыз.',
                'Try another direction.',
              )}
            />
          )}
        </div>
      )}
      <div className="tree-filter">
        <select
          value={filter}
          onChange={(e) => changeFilter(e.target.value)}
          aria-label={tr('Направление развития', 'Даму бағыты', 'Development direction')}
        >
          <option value="">{tr('Все направления', 'Барлық бағыттар', 'All directions')}</option>
          {categories.map((c) => (
            <option key={c} value={c}>
              {labels.category(c)}
            </option>
          ))}
        </select>
        {view === 'tree' && pageCount > 1 && (
          <button
            className="text-button"
            aria-label={tr(
              'Следующие модули на дереве',
              'Ағаштағы келесі модульдер',
              'Next modules on the tree',
            )}
            onClick={() => setPage((p) => (p + 1) % pageCount)}
          >
            {page + 1}/{pageCount} <ArrowRight size={14} />
          </button>
        )}
      </div>
      <div className="energy-row">
        <Zap size={14} />
        <span>{tr('Энергия', 'Қуат', 'Energy')}</span>
        <div className="progress-track">
          <span style={{ width: `${Math.max(0, Math.min(100, experience.energy))}%` }} />
        </div>
        <strong>{experience.energy}/100</strong>
      </div>
      <p className="section-note">
        {tr(
          'Нажмите на яблоко. Каждый подтверждённый результат помогает дереву расти.',
          'Алманы басыңыз. Әр расталған нәтиже ағаштың өсуіне көмектеседі.',
          'Select an apple. Every approved result helps your tree grow.',
        )}
      </p>
    </section>
  );
}
