import { useEffect, useState } from 'react';
import {
  ArrowRight,
  Check,
  CheckCircle2,
  Clock3,
  Filter,
  RotateCcw,
  Search,
  ShieldCheck,
} from 'lucide-react';
import { api, requestKey } from '../api/client';
import type { EmployeeSummary, Session, SubmissionDetail } from '../api/types';
import { formatDate, useI18n } from '../i18n';
import { DecisionHistory, SubmissionProof } from './ModulePanel';
import { Empty, ErrorMessage, Loading, Modal, Status } from './Shared';
function ReviewPanel({
  initial,
  session,
  onClose,
  onChanged,
}: {
  initial: SubmissionDetail;
  session: Session;
  onClose: () => void;
  onChanged: () => void;
}) {
  const { tr, locale } = useI18n();
  const [detail, setDetail] = useState(initial);
  const [comment, setComment] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState('');
  const [revokeMode, setRevokeMode] = useState(false);
  const [keys, setKeys] = useState<Record<string, string>>({});
  const own = session.employee_id === detail.employee.employee_id;
  const waiting = (date: string) => {
    const minutes = Math.max(0, Math.floor((Date.now() - new Date(date).getTime()) / 60000));
    return minutes < 60
      ? `${minutes} ${tr('мин', 'мин', 'min')}`
      : `${Math.floor(minutes / 60)} ${tr('ч', 'сағ', 'h')}`;
  };
  const approval = [...(detail.decisions || [])]
    .reverse()
    .find(
      (d) =>
        d.action === 'approve' &&
        d.submission_id === detail.submission.id &&
        !detail.decisions.some((other) => other.reversal_of === d.id),
    );
  async function decide(action: 'approve' | 'return' | 'revoke') {
    setError('');
    if (action !== 'approve' && !comment.trim()) {
      setError(
        tr(
          'Напишите причину, чтобы сотрудник понимал следующий шаг.',
          'Қызметкер келесі қадамды түсінуі үшін себебін жазыңыз.',
          'Add a reason so the employee knows what to do next.',
        ),
      );
      return;
    }
    setBusy(true);
    const key = keys[action] || requestKey();
    setKeys((prev) => ({ ...prev, [action]: key }));
    try {
      const next =
        action === 'revoke'
          ? await api.revoke(approval!.id, comment, key)
          : await api.decide(detail.submission.id, action, comment, key);
      setDetail(next);
      setComment('');
      setKeys({});
      setRevokeMode(false);
      setNotice(
        action === 'approve'
          ? tr(
              'Результат подтверждён. Навыки и EXP начислены.',
              'Нәтиже расталды. Дағдылар мен EXP есептелді.',
              'Result approved. Skills and EXP have been awarded.',
            )
          : action === 'return'
            ? tr(
                'Задание возвращено на доработку. EXP не начислен.',
                'Тапсырма толықтыруға қайтарылды. EXP есептелмеді.',
                'Changes requested. No EXP has been awarded.',
              )
            : tr(
                'Подтверждение отменено. Прогресс пересчитан, история сохранена.',
                'Растау жойылды. Прогресс қайта есептелді, тарих сақталды.',
                'Approval revoked. Progress was recalculated and the history retained.',
              ),
      );
      onChanged();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal title={detail.event.title} onClose={onClose} wide>
      <div className="row space-between wrap">
        <div>
          <strong>{detail.employee.full_name}</strong>
          <p className="small">
            {detail.employee.grade} {detail.employee.role} · {detail.employee.department}
          </p>
          {detail.submission.status === 'pending' && (
            <p>
              {tr('Ожидает проверки:', 'Тексеруді күтуде:', 'Awaiting review:')}{' '}
              {waiting(detail.submission.submitted_at)}
            </p>
          )}
        </div>
        <Status value={detail.submission.status} />
      </div>
      <section className="dialog-section">
        <h3>{tr('Критерии результата', 'Нәтиже критерийлері', 'Result criteria')}</h3>
        <ul className="criteria">
          {(
            detail.enrollment.config.criteria[locale] ||
            detail.enrollment.config.criteria.ru ||
            []
          ).map((criterion, index) => (
            <li key={index}>{criterion}</li>
          ))}
        </ul>
        <p className="section-note">
          {detail.enrollment.config.reward_exp} EXP ·{' '}
          {tr('Версия задания', 'Тапсырма нұсқасы', 'Task version')}{' '}
          {detail.enrollment.config.version}
        </p>
      </section>
      <section className="dialog-section">
        <h3>{tr('Результат сотрудника', 'Қызметкер нәтижесі', 'Employee result')}</h3>
        <p className="section-note">
          {tr('Отправлено', 'Жіберілді', 'Submitted')}{' '}
          {formatDate(detail.submission.submitted_at, locale, true)} ·{' '}
          {tr('Версия', 'Нұсқа', 'Version')} {detail.submission.version}
        </p>
        <SubmissionProof submission={detail.submission} />
      </section>
      {error && <ErrorMessage message={error} />}{' '}
      {notice && (
        <div className="success" role="status">
          <CheckCircle2 size={19} />
          {notice}
        </div>
      )}
      {own && (
        <div className="info-banner">
          <ShieldCheck size={20} />
          {tr(
            'Нельзя проверять собственный результат. Нужен другой HR.',
            'Өз нәтижеңізді тексеруге болмайды. Басқа HR қажет.',
            'You cannot review your own result. Another HR reviewer is required.',
          )}
        </div>
      )}
      {!own && (detail.submission.status === 'pending' || revokeMode) && (
        <section className="dialog-section">
          <label>
            {revokeMode
              ? tr('Причина отмены', 'Жою себебі', 'Reason for revoking approval')
              : tr('Комментарий сотруднику', 'Қызметкерге пікір', 'Feedback for the employee')}
            <textarea
              value={comment}
              maxLength={4000}
              onChange={(e) => {
                setComment(e.target.value);
                setKeys({});
              }}
              placeholder={tr(
                'Что получилось или что нужно доработать…',
                'Не жақсы шықты немесе нені толықтыру керек…',
                'What worked well or what needs improving…',
              )}
            />
          </label>
          <div className="review-toolbar">
            {revokeMode ? (
              <>
                <button className="button danger" disabled={busy} onClick={() => decide('revoke')}>
                  <RotateCcw size={16} />
                  {tr('Отменить подтверждение', 'Растауды жою', 'Revoke approval')}
                </button>
                <button
                  className="button secondary"
                  disabled={busy}
                  onClick={() => setRevokeMode(false)}
                >
                  {tr('Назад', 'Артқа', 'Back')}
                </button>
              </>
            ) : (
              <>
                <button
                  className="button primary"
                  disabled={busy}
                  onClick={() => decide('approve')}
                >
                  <Check size={17} />
                  {tr('Подтвердить результат', 'Нәтижені растау', 'Approve result')}
                </button>
                <button
                  className="button secondary"
                  disabled={busy}
                  onClick={() => decide('return')}
                >
                  <RotateCcw size={16} />
                  {tr('На доработку', 'Толықтыруға', 'Request changes')}
                </button>
              </>
            )}
          </div>
          <p className="section-note">
            {tr(
              'Для возврата и отмены комментарий обязателен. Решение и начисление сохраняются вместе.',
              'Қайтару және жою үшін пікір міндетті. Шешім мен есептеу бірге сақталады.',
              'A comment is required for changes or revocation. The decision and award are saved together.',
            )}
          </p>
        </section>
      )}
      {!own && detail.submission.status === 'completed' && approval && !revokeMode && (
        <button
          className="text-button"
          onClick={() => {
            setNotice('');
            setRevokeMode(true);
          }}
        >
          {tr(
            'Исправить ошибочное подтверждение',
            'Қате растауды түзету',
            'Correct a mistaken approval',
          )}
          <RotateCcw size={14} />
        </button>
      )}
      {detail.decisions?.length > 0 && (
        <section className="dialog-section">
          <h3>{tr('История решений', 'Шешімдер тарихы', 'Decision history')}</h3>
          <DecisionHistory decisions={detail.decisions} />
        </section>
      )}
      {detail.versions?.length > 1 && (
        <section className="dialog-section">
          <h3>{tr('Предыдущие отправки', 'Алдыңғы жіберулер', 'Previous submissions')}</h3>
          {detail.versions
            .filter((v) => v.id !== detail.submission.id)
            .map((v) => (
              <details className="proof-version" key={v.id}>
                <summary>
                  <span>
                    {tr('Версия', 'Нұсқа', 'Version')} {v.version} ·{' '}
                    {formatDate(v.submitted_at, locale, true)}
                  </span>
                  <Status value={v.status} />
                </summary>
                <div>
                  <SubmissionProof submission={v} />
                </div>
              </details>
            ))}
        </section>
      )}
    </Modal>
  );
}
export default function ReviewQueue({
  employees,
  session,
  onEmployee,
}: {
  employees: EmployeeSummary[];
  session: Session;
  onEmployee: (id: string) => void;
}) {
  const { tr, locale } = useI18n();
  const [items, setItems] = useState<SubmissionDetail[] | null>(null);
  const [eventOptions, setEventOptions] = useState<Record<string, string>>({});
  const [pendingCount, setPendingCount] = useState(0);
  const [status, setStatus] = useState('pending');
  const [employee, setEmployee] = useState('');
  const [eventId, setEventId] = useState('');
  const [query, setQuery] = useState('');
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');
  const [error, setError] = useState('');
  const [revision, setRevision] = useState(0);
  const [selected, setSelected] = useState<SubmissionDetail | null>(null);
  useEffect(() => {
    const abort = new AbortController();
    const params = new URLSearchParams();
    if (status) params.set('status', status);
    if (employee) params.set('employee_id', employee);
    if (eventId) params.set('event_id', eventId);
    if (from) params.set('from', from);
    if (to) params.set('to', to);
    setError('');
    api
      .reviews(params, abort.signal)
      .then((result) => {
        setItems(result.items || []);
        setEventOptions((previous) => ({
          ...previous,
          ...Object.fromEntries(
            (result.items || []).map((item) => [item.event.event_id, item.event.title]),
          ),
        }));
        setPendingCount(result.pending_count);
      })
      .catch((err) => {
        if (!abort.signal.aborted) setError(err.message);
      });
    return () => abort.abort();
  }, [status, employee, eventId, from, to, locale, revision]);
  useEffect(() => {
    const refresh = () => {
      if (document.visibilityState === 'visible') setRevision((v) => v + 1);
    };
    window.addEventListener('focus', refresh);
    const timer = setInterval(refresh, 15000);
    return () => {
      window.removeEventListener('focus', refresh);
      clearInterval(timer);
    };
  }, []);
  const waiting = (date: string) => {
    const minutes = Math.max(0, Math.floor((Date.now() - new Date(date).getTime()) / 60000));
    return minutes < 60
      ? `${minutes} ${tr('мин', 'мин', 'min')}`
      : `${Math.floor(minutes / 60)} ${tr('ч', 'сағ', 'h')}`;
  };
  const filtered = (items || []).filter((item) =>
    `${item.employee.full_name} ${item.event.title} ${item.employee.employee_id}`
      .toLowerCase()
      .includes(query.toLowerCase()),
  );
  const events = Object.entries(eventOptions).sort(([a], [b]) => a.localeCompare(b));
  return (
    <>
      <div className="section-heading">
        <div>
          <h2>
            {tr(
              'Дайте росту зелёный свет',
              'Өсуге жасыл жол ашыңыз',
              'Give growth the green light',
            )}
          </h2>
          <p>
            {tr(
              'Проверьте результат и помогите сотруднику двигаться дальше.',
              'Нәтижені тексеріп, қызметкердің алға жылжуына көмектесіңіз.',
              'Review the result and help the employee move forward.',
            )}
          </p>
        </div>
        <span className="tag amber">
          <Clock3 size={14} />
          {pendingCount} {tr('ожидают', 'күтуде', 'pending')}
        </span>
      </div>
      <div className="filters">
        <label className="search">
          <Search size={16} />
          <input
            aria-label={tr('Поиск заявок', 'Өтінімдерді іздеу', 'Search submissions')}
            placeholder={tr(
              'Сотрудник или модуль…',
              'Қызметкер немесе модуль…',
              'Employee or module…',
            )}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </label>
        <select
          aria-label={tr('Статус заявки', 'Өтінім күйі', 'Submission status')}
          value={status}
          onChange={(e) => setStatus(e.target.value)}
        >
          <option value="">{tr('Все статусы', 'Барлық күйлер', 'All statuses')}</option>
          <option value="pending">{tr('На проверке', 'Тексерілуде', 'Awaiting review')}</option>
          <option value="changes_requested">
            {tr('На доработке', 'Толықтыруда', 'Changes requested')}
          </option>
          <option value="completed">{tr('Подтверждены', 'Расталған', 'Approved')}</option>
          <option value="revoked">{tr('Отменены', 'Жойылған', 'Revoked')}</option>
        </select>
        <select
          aria-label={tr('Сотрудник', 'Қызметкер', 'Employee')}
          value={employee}
          onChange={(e) => setEmployee(e.target.value)}
        >
          <option value="">{tr('Все сотрудники', 'Барлық қызметкерлер', 'All employees')}</option>
          {employees.map((item) => (
            <option key={item.employee_id} value={item.employee_id}>
              {item.full_name}
            </option>
          ))}
        </select>
        <button
          className="icon-button"
          aria-label={tr('Обновить очередь', 'Кезекті жаңарту', 'Refresh queue')}
          onClick={() => setRevision((v) => v + 1)}
        >
          <RotateCcw size={18} />
        </button>
      </div>
      <details className="section-note" style={{ marginBottom: 20 }}>
        <summary className="text-button">
          <Filter size={14} />
          {tr('Модуль и дата отправки', 'Модуль және жіберу күні', 'Module and submission date')}
        </summary>
        <div className="filters">
          <select
            aria-label={tr('Модуль', 'Модуль', 'Module')}
            value={eventId}
            onChange={(e) => setEventId(e.target.value)}
          >
            <option value="">{tr('Все модули', 'Барлық модульдер', 'All modules')}</option>
            {events.map(([id, title]) => (
              <option key={id} value={id}>
                {title}
              </option>
            ))}
          </select>
          <label>
            {tr('С', 'Бастап', 'From')}
            <input type="date" value={from} onChange={(e) => setFrom(e.target.value)} />
          </label>
          <label>
            {tr('По', 'Дейін', 'To')}
            <input type="date" value={to} min={from} onChange={(e) => setTo(e.target.value)} />
          </label>
        </div>
      </details>
      {error && <ErrorMessage message={error} retry={() => setRevision((v) => v + 1)} />}{' '}
      {!items && !error && <Loading />}{' '}
      {items && (
        <div className="queue-list">
          {filtered.map((detail) => (
            <article className="queue-card panel" key={detail.submission.id}>
              <span className="mini-avatar">
                {detail.employee.full_name
                  .split(' ')
                  .slice(0, 2)
                  .map((part) => part[0])
                  .join('')}
              </span>
              <div>
                <button
                  className="employee-link"
                  onClick={() => onEmployee(detail.employee.employee_id)}
                >
                  {detail.employee.full_name}
                </button>
                <h3>{detail.event.title}</h3>
                <p>
                  {formatDate(detail.submission.submitted_at, locale, true)} ·{' '}
                  {tr('Версия', 'Нұсқа', 'Version')} {detail.submission.version} ·{' '}
                  {detail.enrollment.config.reward_exp} EXP
                </p>
                {detail.submission.status === 'pending' && (
                  <p>
                    {tr('Ожидает проверки:', 'Тексеруді күтуде:', 'Awaiting review:')}{' '}
                    {waiting(detail.submission.submitted_at)}
                  </p>
                )}
              </div>
              <Status value={detail.submission.status} />
              <button className="button secondary small" onClick={() => setSelected(detail)}>
                {detail.submission.status === 'pending'
                  ? tr('Проверить', 'Тексеру', 'Review')
                  : tr('Открыть', 'Ашу', 'Open')}
                <ArrowRight size={15} />
              </button>
            </article>
          ))}
          {!filtered.length && (
            <div className="panel">
              <Empty
                title={tr('Все спокойно', 'Бәрі тыныш', 'All caught up')}
                text={tr(
                  'По этим фильтрам заявок нет. Новые результаты появятся здесь автоматически.',
                  'Бұл сүзгілер бойынша өтінімдер жоқ. Жаңа нәтижелер осында автоматты түрде пайда болады.',
                  'No submissions match these filters. New results will appear here automatically.',
                )}
              />
            </div>
          )}
        </div>
      )}
      {selected && (
        <ReviewPanel
          initial={selected}
          session={session}
          onClose={() => setSelected(null)}
          onChanged={() => setRevision((v) => v + 1)}
        />
      )}
    </>
  );
}
