import { useEffect, useRef, useState, type FormEvent } from 'react';
import {
  ArrowRight,
  CheckCircle2,
  Clock3,
  ExternalLink,
  FileText,
  Paperclip,
  Send,
  ShieldCheck,
  Sparkles,
  Zap,
} from 'lucide-react';
import { api, requestKey } from '../api/client';
import type {
  AssistantAction,
  AssistantResponse,
  EnrollmentDetail,
  ModuleView,
  Submission,
  ReviewDecision,
} from '../api/types';
import { formatDate, useI18n, useLabels } from '../i18n';
import { ErrorMessage, Loading, Modal, Status } from './Shared';
export function SubmissionProof({ submission }: { submission: Submission }) {
  const { tr } = useI18n();
  return (
    <div>
      {submission.text && <p className="evidence-text">{submission.text}</p>}
      {/^https:\/\//i.test(submission.url) && (
        <a
          className="evidence-link"
          href={submission.url}
          target="_blank"
          rel="noopener noreferrer"
        >
          <ExternalLink size={15} />
          {submission.url}
        </a>
      )}
      {(submission.attachments || []).map((file) => (
        <a
          className="file-chip"
          key={file.id}
          href={`/api/attachments/${encodeURIComponent(file.id)}`}
          target="_blank"
          rel="noopener noreferrer"
        >
          <Paperclip size={15} />
          <span>{file.filename}</span>
          <small className="muted">{Math.ceil(file.size / 1024)} KB</small>
          <span className="muted">{tr('Скачать', 'Жүктеу', 'Download')}</span>
        </a>
      ))}
    </div>
  );
}
export function DecisionHistory({ decisions }: { decisions: ReviewDecision[] }) {
  const { tr, locale } = useI18n();
  return (
    <div className="review-history">
      {decisions.map((item) => (
        <div key={item.id}>
          <strong className="small">
            {item.action === 'approve'
              ? tr('HR подтвердил результат', 'HR нәтижені растады', 'HR approved the result')
              : item.action === 'return'
                ? tr('HR вернул на доработку', 'HR толықтыруға қайтарды', 'HR requested changes')
                : tr('Подтверждение отменено', 'Растау жойылды', 'Approval revoked')}
          </strong>
          <small className="block">
            {formatDate(item.recorded_at, locale, true)} ·{' '}
            {item.reviewer_login || tr('HR-команда', 'HR командасы', 'HR team')}
          </small>
          {item.comment && <p>{item.comment}</p>}
        </div>
      ))}
    </div>
  );
}
export function Assistant({
  employeeId,
  eventId,
  onAlternative,
}: {
  employeeId: string;
  eventId: string;
  onAlternative?: (id: string) => void;
}) {
  const { tr, locale } = useI18n();
  const [result, setResult] = useState<AssistantResponse | null>(null);
  const [active, setActive] = useState<AssistantAction | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const generation = useRef(0);
  useEffect(() => {
    generation.current++;
    setResult(null);
    setError('');
    setActive(null);
    setBusy(false);
    return () => {
      generation.current++;
    };
  }, [eventId, locale]);
  const actions: [AssistantAction, string][] = [
    ['why', tr('Зачем мне это?', 'Бұл маған не үшін?', 'Why does this matter?')],
    ['fifteen_minutes', tr('Есть 15 минут', '15 минутым бар', 'I have 15 minutes')],
    ['simplify', tr('Слишком сложно', 'Тым қиын', 'Make it simpler')],
    ['alternative', tr('Другой вариант', 'Басқа нұсқа', 'Another option')],
    ['start', tr('Помоги начать', 'Бастауға көмектес', 'Help me start')],
  ];
  async function ask(action: AssistantAction) {
    const turn = ++generation.current;
    setBusy(true);
    setActive(action);
    setError('');
    try {
      const answer = await api.assistant(employeeId, eventId, action, locale);
      if (turn === generation.current) setResult(answer);
    } catch (err) {
      if (turn === generation.current) setError((err as Error).message);
    } finally {
      if (turn === generation.current) setBusy(false);
    }
  }
  return (
    <section className="assistant-box">
      <div className="assistant-heading">
        <Sparkles size={18} />
        {tr('Ваш помощник в развитии', 'Дамудағы көмекшіңіз', 'Your growth companion')}
      </div>
      <div className="assistant-actions">
        {actions.map(([action, label]) => (
          <button
            key={action}
            className={active === action ? 'active' : ''}
            disabled={busy}
            onClick={() => ask(action)}
          >
            {label}
          </button>
        ))}
      </div>
      {busy && (
        <Loading
          text={tr(
            'Ищем полезный первый шаг…',
            'Пайдалы алғашқы қадамды іздейміз…',
            'Finding a useful first step…',
          )}
        />
      )}{' '}
      {error && <ErrorMessage message={error} />}
      {result && !busy && (
        <div className="assistant-answer" role="status">
          <div>
            <h4>{tr('Зачем вам это', 'Бұл не үшін керек', 'Why it matters to you')}</h4>
            <p>{result.benefit}</p>
          </div>
          <div>
            <h4>{tr('С чего начать', 'Неден бастау керек', 'Where to start')}</h4>
            <p>{result.first_step}</p>
          </div>
          <div>
            <h4>{tr('Применение в работе', 'Жұмыста қолдану', 'Put it into practice')}</h4>
            <p>{result.application}</p>
          </div>
          {result.alternative_event_id && onAlternative && (
            <button
              className="text-button"
              onClick={() => onAlternative(result.alternative_event_id!)}
            >
              {tr('Открыть альтернативу', 'Басқа нұсқаны ашу', 'Open alternative')}
              <ArrowRight size={14} />
            </button>
          )}
          <small className="muted">
            {result.mode === 'llm'
              ? tr('Персональный ответ AI', 'AI жеке жауабы', 'Personal AI guidance')
              : tr(
                  'Подсказка по правилам · AI недоступен',
                  'Ережеге негізделген кеңес · AI қолжетімсіз',
                  'Rule-based guidance · AI unavailable',
                )}
            {result.notice ? ` · ${result.notice}` : ''}
          </small>
          <p className="small">
            {tr(
              'Практическая подсказка — не отдельное задание и не даёт EXP сама по себе.',
              'Практикалық кеңес жеке тапсырма емес және өздігінен EXP бермейді.',
              'Practice guidance is not a separate assessed task and does not award EXP.',
            )}
          </p>
          {result.evidence?.length > 0 && (
            <details>
              <summary className="small">
                {tr(
                  'На каких фактах основано',
                  'Қандай деректерге негізделген',
                  'Supporting facts',
                )}
              </summary>
              <ul className="criteria">
                {result.evidence.map((f) => (
                  <li key={f.id}>{f.text}</li>
                ))}
              </ul>
            </details>
          )}
        </div>
      )}
    </section>
  );
}
export default function ModulePanel({
  module,
  employeeId,
  readOnly = false,
  onClose,
  onChanged,
  onAlternative,
}: {
  module: ModuleView;
  employeeId: string;
  readOnly?: boolean;
  onClose: () => void;
  onChanged: () => void;
  onAlternative: (id: string) => void;
}) {
  const { tr, locale } = useI18n();
  const labels = useLabels();
  const [detail, setDetail] = useState<EnrollmentDetail | null>(null);
  const [loading, setLoading] = useState(!!module.enrollment);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState('');
  const [text, setText] = useState('');
  const [url, setUrl] = useState('');
  const [files, setFiles] = useState<File[]>([]);
  const submitKey = useRef('');
  useEffect(() => {
    let active = true;
    if (module.enrollment)
      api
        .enrollment(module.enrollment.id)
        .then((value) => {
          if (active) setDetail(value);
        })
        .catch((err) => {
          if (active) setError(err.message);
        })
        .finally(() => {
          if (active) setLoading(false);
        });
    return () => {
      active = false;
    };
  }, [module.enrollment?.id, module.state]);
  const state = detail?.enrollment.state || module.state;
  const config = detail?.enrollment.config || module.config;
  const criteria = config.criteria[locale] || config.criteria.ru || [];
  async function start() {
    setBusy(true);
    setError('');
    try {
      setDetail(await api.start(employeeId, module.event.event_id));
      onChanged();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }
  async function submit(event: FormEvent) {
    event.preventDefault();
    setError('');
    if (!detail) return;
    if (!text.trim() && !url.trim() && !files.length) {
      setError(
        tr(
          'Добавьте описание результата, ссылку или файл.',
          'Нәтиже сипаттамасын, сілтемені немесе файлды қосыңыз.',
          'Add a result description, link or file.',
        ),
      );
      return;
    }
    if (url && !/^https:\/\//i.test(url)) {
      setError(
        tr(
          'Ссылка должна начинаться с https://',
          'Сілтеме https:// арқылы басталуы керек',
          'Links must start with https://',
        ),
      );
      return;
    }
    if (
      files.length > 3 ||
      files.some(
        (file) =>
          file.size > 5 * 1024 * 1024 ||
          !['application/pdf', 'image/png', 'image/jpeg'].includes(file.type),
      )
    ) {
      setError(
        tr(
          'Можно приложить до 3 файлов: PDF, PNG или JPEG, каждый до 5 МБ.',
          'Әрқайсысы 5 МБ дейінгі 3 файлға дейін: PDF, PNG немесе JPEG.',
          'Attach up to 3 PDF, PNG or JPEG files, up to 5 MB each.',
        ),
      );
      return;
    }
    setBusy(true);
    if (!submitKey.current) submitKey.current = requestKey();
    const data = new FormData();
    data.set('text', text.trim());
    data.set('url', url.trim());
    data.set('request_key', submitKey.current);
    files.forEach((file) => data.append('files', file));
    try {
      const saved = await api.submit(detail.enrollment.id, data);
      setDetail(saved);
      setNotice(
        tr(
          'Результат отправлен HR. Мы обновим прогресс после подтверждения.',
          'Нәтиже HR-ға жіберілді. Расталғаннан кейін прогресс жаңартылады.',
          'Your result is with HR. Progress will update after approval.',
        ),
      );
      setText('');
      setUrl('');
      setFiles([]);
      submitKey.current = '';
      onChanged();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal title={module.event.title} onClose={onClose} wide>
      <div className="row wrap space-between">
        <Status value={state} />
        <div className="module-meta">
          <span>
            <Clock3 size={14} />
            {module.event.duration_hours} {tr('ч', 'сағ', 'h')}
          </span>
          <span>{labels.format(module.event.format)}</span>
          <span>
            <Zap size={14} />
            {config.reward_exp} EXP
          </span>
        </div>
      </div>
      {module.event.description && (
        <section className="dialog-section">
          <h3>{tr('О модуле', 'Модуль туралы', 'About this module')}</h3>
          <p className="module-description">{module.event.description}</p>
        </section>
      )}
      <section className="dialog-section">
        <h3>{tr('Что вы получите', 'Сіз не аласыз', 'What you will achieve')}</h3>
        <p className="module-outcome">
          {config.outcome[locale] || config.outcome.ru || module.event.description}
        </p>
        <div className="gain-pills">
          {(module.expected_gains || []).map((g) => (
            <span className="gain-pill" key={g.skill_id}>
              {g.name}{' '}
              <strong>
                {g.before} → {g.after}
              </strong>
            </span>
          ))}
        </div>
      </section>
      <section className="dialog-section">
        <h3>
          {tr(
            'Один результат для проверки',
            'Тексеруге арналған бір нәтиже',
            'One result to submit',
          )}
        </h3>
        <ul className="criteria">
          {criteria.map((criterion, index) => (
            <li key={index}>{criterion}</li>
          ))}
        </ul>
        <p className="section-note">
          <ShieldCheck size={13} />{' '}
          {tr(
            'Навыки и EXP начисляются только после подтверждения HR. Оценка дерева не означает повышение грейда.',
            'Дағдылар мен EXP тек HR растағаннан кейін есептеледі. Ағаш деңгейі қызметтік деңгейдің өсуін білдірмейді.',
            'Skills and EXP are awarded only after HR approval. Tree level does not mean a promotion.',
          )}
        </p>
      </section>
      <Assistant
        employeeId={employeeId}
        eventId={module.event.event_id}
        onAlternative={onAlternative}
      />
      {loading && <Loading />}
      {error && <ErrorMessage message={error} />}{' '}
      {notice && (
        <div className="success" role="status">
          <CheckCircle2 size={19} />
          {notice}
        </div>
      )}
      {state === 'locked' && (
        <div className="info-banner">
          {module.blocked_reason ||
            tr(
              'Сначала выполните условия доступа к модулю.',
              'Алдымен модульге кіру талаптарын орындаңыз.',
              'Complete the prerequisites first.',
            )}
        </div>
      )}
      {!readOnly && state === 'available' && (
        <div className="form-actions">
          <button className="button primary" disabled={busy} onClick={start}>
            {busy
              ? tr('Начинаем…', 'Басталуда…', 'Starting…')
              : tr('Начать модуль', 'Модульді бастау', 'Start module')}
            <ArrowRight size={16} />
          </button>
        </div>
      )}
      {state === 'pending' && (
        <div className="dialog-section pending-note">
          <Clock3 size={20} />
          <p>
            {tr(
              'Результат на проверке. Пока HR не подтвердит его, навыки и EXP остаются прежними.',
              'Нәтиже тексерілуде. HR растағанға дейін дағдылар мен EXP өзгермейді.',
              'Your result is awaiting review. Skills and EXP stay unchanged until HR approves it.',
            )}
          </p>
        </div>
      )}
      {detail && detail.decisions?.length > 0 && <DecisionHistory decisions={detail.decisions} />}
      {!readOnly && detail && (state === 'in_progress' || state === 'changes_requested') && (
        <section className="dialog-section">
          <h3>{tr('Покажите свой результат', 'Нәтижеңізді көрсетіңіз', 'Share your result')}</h3>
          <form className="proof-form" onSubmit={submit}>
            <label>
              {tr(
                'Что вы сделали и чему научились',
                'Не істедіңіз және не үйрендіңіз',
                'What you did and learned',
              )}
              <textarea
                maxLength={10000}
                value={text}
                onChange={(e) => {
                  setText(e.target.value);
                  submitKey.current = '';
                }}
                placeholder={tr(
                  'Опишите результат и его применение в работе…',
                  'Нәтижені және оны жұмыста қолдануды сипаттаңыз…',
                  'Describe your result and how you will use it at work…',
                )}
              />
            </label>
            <label>
              {tr('Ссылка на результат', 'Нәтижеге сілтеме', 'Link to your result')}
              <input
                type="url"
                maxLength={2048}
                value={url}
                placeholder="https://…"
                onChange={(e) => {
                  setUrl(e.target.value);
                  submitKey.current = '';
                }}
              />
              <small>
                {tr(
                  'Дайте HR доступ к документу по ссылке.',
                  'HR-ға сілтемедегі құжатқа қол жеткізуге мүмкіндік беріңіз.',
                  'Make sure HR can access the linked document.',
                )}
              </small>
            </label>
            <label className="file-zone">
              <span className="row">
                <Paperclip size={17} />
                {tr('Приложить доказательства', 'Дәлелдерді тіркеу', 'Attach evidence')}
              </span>
              <input
                type="file"
                accept=".pdf,.png,.jpg,.jpeg,application/pdf,image/png,image/jpeg"
                multiple
                onChange={(e) => {
                  setFiles(Array.from(e.target.files || []));
                  submitKey.current = '';
                }}
              />
              <small>
                {tr(
                  'До 3 файлов по 5 МБ: PDF, PNG, JPEG. Видны только вам и HR.',
                  '5 МБ дейінгі 3 файл: PDF, PNG, JPEG. Тек сізге және HR-ға көрінеді.',
                  'Up to 3 files, 5 MB each: PDF, PNG, JPEG. Only you and HR can access them.',
                )}
              </small>
            </label>
            <button className="button primary full" disabled={busy}>
              {busy
                ? tr('Отправляем…', 'Жіберілуде…', 'Sending…')
                : tr('Отправить на проверку', 'Тексеруге жіберу', 'Submit for review')}
              <Send size={16} />
            </button>
          </form>
        </section>
      )}
      {detail && detail.versions?.length > 0 && (
        <section className="dialog-section">
          <h3>
            <FileText size={15} /> {tr('История отправок', 'Жіберу тарихы', 'Submission history')}
          </h3>
          {[...detail.versions].reverse().map((version) => (
            <details className="proof-version" key={version.id}>
              <summary>
                <span>
                  {tr('Версия', 'Нұсқа', 'Version')} {version.version} ·{' '}
                  {formatDate(version.submitted_at, locale, true)}
                </span>
                <Status value={version.status} />
              </summary>
              <div>
                <SubmissionProof submission={version} />
              </div>
            </details>
          ))}
        </section>
      )}
      {readOnly && (
        <p className="section-note">
          {tr(
            'Вы просматриваете профиль сотрудника. Решения по заявкам доступны в разделе «Проверки».',
            'Сіз қызметкер профилін қарап отырсыз. Өтінімдер бойынша шешімдер «Тексерулер» бөлімінде қолжетімді.',
            'You are viewing an employee profile. Review submissions in the Reviews section.',
          )}
        </p>
      )}
    </Modal>
  );
}
