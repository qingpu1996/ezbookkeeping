import { describe, it, expect } from 'vitest';
import { investmentMinorUnits, investmentMoney, investmentAverage } from './investment.ts';
describe('investment exact display amounts', () => {
    it('parses decimal amounts without floating point error', () => {
        expect(investmentMinorUnits('0.29')).toBe('29');
        expect(investmentMinorUnits('4805')).toBe('480500');
        expect(investmentMoney('-23400')).toBe('-234.00');
        expect(investmentAverage('1380500', '15')).toBe('920.3333');
        expect(investmentAverage('0', '0')).toBe('—');
    });
    it.each(['NaN', '-1', '1e2', '1.001', '9999999999999'])('rejects invalid amount %s', value => {
        expect(() => investmentMinorUnits(value)).toThrow();
    });
});
