import { useInfiniteQuery } from '@tanstack/react-query';
import { queryKeys } from '@/lib/query-keys';

export interface BrainrotItem {
    id: string;
    type: 'image' | 'video';
    url: string;
    title?: string;
    aspectRatio: number;
}

const MOCK_IMAGES = [
    'https://pbs.twimg.com/media/GFPI8LKXQAAis33.jpg',
    'https://ih1.redbubble.net/image.5681776202.5017/st,small,507x507-pad,600x600,f8f8f8.jpg',
    'https://media.tenor.com/kIjcbnIniMsAAAAe/sukuna.png',
    'https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcTIQ40UACBhEOuo_xTJAAq3OCP08GVt7n3Dwg&s',
    'https://i.pinimg.com/736x/bf/6e/29/bf6e296386c67b027cd3d234e3c6efa4.jpg',
    'https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcQ6Gtano8OnHkaq-6hB7G-ts9KVw99Ye5ReSw&s',
    'https://media.tenor.com/uKayqry3x90AAAAe/goofy-funny-cat.png',
    'https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcRyEuhvF2TrAx1vYVTcQ1N8QpRYPrpj7qHwoA&s',
    'https://media.tenor.com/uKayqry3x90AAAAe/goofy-funny-cat.png',
    'https://media.tenor.com/5E08lV96ikAAAAAe/sillycat.png',
    'https://i.pinimg.com/236x/28/df/05/28df0588533fcaa59cd3f0ca45b56d3c.jpg',
    'https://media.tenor.com/owsPz6f26FcAAAAM/happy-cat-silly-cat.gif',
];

async function fetchBrainrot(page: number): Promise<{ items: BrainrotItem[]; nextPage: number }> {
    // No simulated delay for better UX

    const items: BrainrotItem[] = Array.from({ length: 10 }).map((_, i) => {
        const index = (page * 10 + i) % MOCK_IMAGES.length;
        return {
            id: `brainrot-${page}-${i}-${Date.now()}`,
            type: 'image',
            url: MOCK_IMAGES[index],
            aspectRatio: 1,
        };
    });

    return {
        items,
        nextPage: page + 1,
    };
}

export function useBrainrotFeed() {
    return useInfiniteQuery({
        queryKey: queryKeys.brainrot,
        queryFn: ({ pageParam }) => fetchBrainrot(pageParam),
        initialPageParam: 0,
        getNextPageParam: (lastPage) => lastPage.nextPage,
    });
}
