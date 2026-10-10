import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { formatFullTime, formatRelativeTime } from '@/lib/format';
import { cn } from '@/lib/utils';
import type { MessageResponse } from '@/types';

/** One DM. Mine are blue on the right; theirs grey on the left. Content is Markdown. */
export function MessageBubble({ message, mine }: { message: MessageResponse; mine: boolean }) {
  return (
    <div className={cn('flex max-w-[85%] flex-col md:max-w-[75%]', mine ? 'items-end self-end' : 'items-start self-start')}>
      <div
        className={cn(
          'max-w-full break-words rounded-3xl px-4 py-2.5 text-[15px] leading-5 [&_a]:underline [&_p]:m-0',
          mine ? 'rounded-br-md bg-primary text-primary-foreground' : 'rounded-bl-md bg-muted text-foreground',
        )}
      >
        <ReactMarkdown
          remarkPlugins={[remarkGfm]}
          components={{
            a: ({ href, children }) => (
              <a href={href} target="_blank" rel="noopener noreferrer nofollow">
                {children}
              </a>
            ),
          }}
        >
          {message.content}
        </ReactMarkdown>
      </div>
      <time dateTime={message.createdAt} title={formatFullTime(message.createdAt)} className="mt-1 px-1 text-[13px] text-muted-foreground" suppressHydrationWarning>
        {formatRelativeTime(message.createdAt)}
      </time>
    </div>
  );
}
