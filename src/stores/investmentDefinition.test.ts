import {beforeEach,describe,it,expect,vi} from 'vitest';
import {createPinia,setActivePinia} from 'pinia';
import services from '@/lib/services';
import {useInvestmentDefinitionsStore} from './investmentDefinition';
vi.mock('@/lib/services',()=>({default:{getInvestmentDefinitions:vi.fn(),saveInvestmentDefinition:vi.fn()}}));
beforeEach(()=>setActivePinia(createPinia()));
describe('definition session cache',()=>{
 it('coalesces reads and keeps fresh memory data',async()=>{vi.mocked(services.getInvestmentDefinitions).mockResolvedValue({data:{result:[]}} as never);const s=useInvestmentDefinitionsStore();await Promise.all([s.load(),s.load()]);await s.load();expect(services.getInvestmentDefinitions).toHaveBeenCalledTimes(1);});
 it('does not restore a previous session after reset',async()=>{let resolve!:(x:never)=>void;vi.mocked(services.getInvestmentDefinitions).mockReturnValue(new Promise(r=>resolve=r));const s=useInvestmentDefinitionsStore();const p=s.load();s.reset();resolve({data:{result:[{id:'private-old-user'}]}} as never);await p;expect(s.rows).toEqual([]);expect(s.loaded).toBe(false);});
});
